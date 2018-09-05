/*
Copyright 2018 BlackRock, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/


import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import io.fabric8.kubernetes. client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClientException;
import io.fabric8.kubernetes.client.Watcher;
import io.fabric8.kubernetes.api.model.ConfigMap;
import org.quartz.*;
import org.quartz.impl.StdScheduler;
import org.quartz.impl.StdSchedulerFactory;
import org.quartz.spi.MutableTrigger;
import org.springframework.scheduling.quartz.SchedulerFactoryBean;
import org.dummy.HolidayCalendar

import javax.net.ssl.HttpsURLConnection;
import java.io.DataOutputStream;
import java.io.File;
import java.net.URL;
import java.util.Date;
import java.util.HashSet;
import java.util.Map;

class Schedule {
    private String calendar;
    private String pattern;
    private String name;

    public Schedule(String calendar, String pattern, String name) {
        this.calendar = calendar;
        this.pattern = pattern;
        this.name = name;
    }

    @Override
    public boolean equals(Object obj) {
        Schedule other = (Schedule)obj;
        if(this.calendar == other.calendar && this.pattern == other.pattern) {
            return true;
        } else {
            return false;
        }
    }

    @Override
    public String toString() {
        return "pattern->" + this.pattern + " calendar->" + this.calendar;
    }

    public String getCalendar() {
        return calendar;
    }

    public void setCalendar(String calendar) {
        this.calendar = calendar;
    }

    public String getPattern() {
        return pattern;
    }

    public void setPattern(String pattern) {
        this.pattern = pattern;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }
}

class Config {
    public static String CONFIG_MAP = "calendar-gateway-configmap";
    public static String NamespaceEnv = "ARGO_EVENTS_NAMESPACE";
    public static String TransformerPortEnv = "TRANSFORMER_PORT";

    private KubernetesClient kubeClient;
    private String namespace;
    public static String transformerPort;
    private HashSet<Schedule> schedules;


    public Config(KubernetesClient client, String namespace, String transformerPort, HashSet<Schedule> schedules) {
        this.kubeClient = client;
        this.namespace = namespace;
        this.transformerPort = transformerPort;
        this.schedules = schedules;
    }

    public KubernetesClient getKubeClient() {
        return kubeClient;
    }

    public void setKubeClient(KubernetesClient kubeClient) {
        this.kubeClient = kubeClient;
    }

    public String getNamespace() {
        return namespace;
    }

    public void setNamespace(String namespace) {
        this.namespace = namespace;
    }

    public HashSet<Schedule> getSchedules() {
        return schedules;
    }

    public void setSchedules(HashSet<Schedule> schedules) {
        this.schedules = schedules;
    }

    public String getTransformerPort() {
        return transformerPort;
    }

    public void setTransformerPort(String transformerPort) {
        this.transformerPort = transformerPort;
    }
}

class ScheduleRunner implements  Runnable {

    private Schedule schedule;

    HolidayCalendar holidayCalendar;

    public HolidayCalendar getHolidayCalendar() {
        return holidayCalendar;
    }

    public void setHolidayCalendar(HolidayCalendar holidayCalendar) {
        this.holidayCalendar = holidayCalendar;
    }

    class ScheduleJob implements Job {

        public void execute(JobExecutionContext context) throws JobExecutionException {

            // Simply check that job time is not a holiday
            Date date = context.getFireTime();

            if(holidayCalendar.isHoliday(date)) {
                // don't post the event as its a holiday
                System.out.println("today is holiday, won't be firing event.");
            } else {
                 try {
                     // post the event to gateway transformer
                     String url = "https://localhost:" + Config.transformerPort;
                     URL obj = new URL(url);
                     HttpsURLConnection con = (HttpsURLConnection) obj.openConnection();

                     //add reuqest header
                     con.setRequestMethod("POST");

                     // Send post request
                     con.setDoOutput(true);
                     DataOutputStream wr = new DataOutputStream(con.getOutputStream());
                     // write an actual payload here.
                     wr.writeInt(0);
                     wr.flush();
                     wr.close();
                 } catch (Exception e){
                     e.printStackTrace();
                }
            }
        }
    }

    public ScheduleRunner(Schedule schedule) {
        this.schedule = schedule;
    }

    public void run() {
        try {
            JobDetail job = JobBuilder.newJob(ScheduleJob.class).withIdentity(schedule.getName()).build();

            // Create the trigger
            Trigger trigger = TriggerBuilder.newTrigger().withIdentity("schedule-name", schedule.getName()).withSchedule(CronScheduleBuilder.cronSchedule(schedule.getPattern())).build();

            Scheduler scheduler = new StdSchedulerFactory().getScheduler("scheduler-for-" + schedule.getName());
            scheduler.start();
            Date d = scheduler.scheduleJob(job, trigger);

        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}

class ConfigWatcher implements Runnable {

    private Config config;

    public ConfigWatcher(Config config) {
        this.config = config;
    }

    public void run() {
        config.getKubeClient().configMaps().inNamespace(config.getNamespace()).withName(Config.CONFIG_MAP).watch(new Watcher<io.fabric8.kubernetes.api.model.ConfigMap>() {
            @Override
            public void eventReceived(Action action, io.fabric8.kubernetes.api.model.ConfigMap configMap) {
                for(Map.Entry<String, String> data: configMap.getData().entrySet()) {
                    String configName = data.getKey();
                    String configData = data.getValue();
                    ObjectMapper mapper = new ObjectMapper(new YAMLFactory());
                    try {
                        Schedule schedule = mapper.readValue(configData, Schedule.class);
                        schedule.setName(configName);
                        if(config.getSchedules().contains(schedule)) {
                            System.out.println("duplicate configuration " + schedule.toString());
                        } else {
                            System.out.println("adding new configuration " + schedule.toString());
                            Thread runnerThread = new Thread(new ScheduleRunner(schedule));
                            runnerThread.start();
                        }
                    } catch (Exception e) {
                        e.printStackTrace();
                    }
                }
            }

            @Override
            public void onClose(KubernetesClientException e) {
                System.out.println("configmap watch has ended. Won't be able to add new configurations");
            }
        });
    }
}

public class CalendarGateway {
    public static void main(String[] args) {
        KubernetesClient client = DefaultKubernetesClient.fromConfig("/Users/vpage/.kube/config");
        String namespace = System.getenv(Config.NamespaceEnv);
        String transformerPort = System.getenv(Config.TransformerPortEnv);
        Config config = new Config(client, namespace, transformerPort, new HashSet<Schedule>());
        ConfigWatcher cw = new ConfigWatcher(config);
        Thread t = new Thread(cw);
        t.start();
    }
}
