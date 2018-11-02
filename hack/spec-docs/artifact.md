












# Argo-Events



Version: 1.0












## Paths




## Definitions


  
### artifact.S3Artifact{#artifact.S3Artifact}


  
  
    
  - insecure *(boolean)*: Mode of operation for s3 client
    


    
  
  
    
  - s3EventConfig\*: S3EventConfig contains configuration for bucket notification
    


    
  

  
### artifact.S3EventConfig{#artifact.S3EventConfig}


  
  
    
  - bucket *(string)*
    


    
  
  
    
  - endpoint *(string)*
    


    
  
  
    
  - [artifact.S3Filter](#artifact.S3Filter)
    


    
  
  
    
  - region *(string)*
    


    
  

  
### artifact.S3Filter{#artifact.S3Filter}


  
  
    
  - prefix\* *(string)*
    


    
  
  
    
  - suffix\* *(string)*
    


    
  

  
