# Salt-API oauth2 middleware

> allow salt-api authentication by using [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/).
> it does a challenged base on trusted header like `X-Forwarded-User`

## SaltStack master example config 

```yaml
rest_cherrypy:
  port: 8000
  host: 127.0.0.1
  disable_ssl: True

external_auth:
  rest:
    '^url': http://127.0.0.1:8080/_challenge
    '*':
      - 'state.*'
      - 'grains.*'
      - 'system.reboot'
      - 'test.ping'
      - 'saltutil.refresh_grains'
      - '@runner'
      - '@wheel'
      - '@jobs'
```

## Auth flow

```
                                                                                        
                                                                                        
                                           +------------------+                         
                               /login      | Salt-API oauth2  |                         
                                   ------->| middleware       |<------                  
                                   |       +------------------+       |                 
                                   |                |                 |                 
            +--------------+       |                |                 |                 
 Client---- |reverse-proxy |------>|                |/login           |/_callback       
            +--------------+       |                |                 |                 
                                   |               \|/                |                 
                                   |           +-----------+          |                 
                                   ----------->| salt-api  |-----------                 
                                               +-----------+                            
                                                                                        
                                                                                        
                                                                                                                                                                                                                                                        
```