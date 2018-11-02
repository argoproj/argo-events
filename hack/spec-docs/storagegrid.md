












# Argo-Events



Version: 1.0












## Paths




## Definitions


  
### storagegrid.Filter{#storagegrid.Filter}


  
  
    
  - prefix\* *(string)*
    


    
  
  
    
  - suffix\* *(string)*
    


    
  

  
### storagegrid.StorageGridEventConfig{#storagegrid.StorageGridEventConfig}


  
  
    
  - endpoint\* *(string)*: Endpoint to listen to events on
    


    
  
  
    
  - events *(array)*: Events are s3 bucket notification events. For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations Note that storage grid notifications do not contain `s3:`
    


    
    
      

      
    - type: string
      

      


    
  
  
    
  - filter: Filter on object key which caused the notification.
    


    
  
  
    
  - port\* *(string)*: Port to run web server on
    


    
  

  
