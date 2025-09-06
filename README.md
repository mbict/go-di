# DI a simplified Dependency Injection container and service locator for go

This is a simplified implementation of a Dependency Injection container and service locator for go.
Under the hood its a simple layer over the uber.Dig package, with some generic helpers.

### Why is created this package?
For most of my projects I need a simple DI container to keep the dependencies manageable, code simple and clear.
This package will also expose a Register method to register requests for the RequestBus package (https://github.com/mbict/requestbus)

### Example usage

```go
    container := di.NewContainer(func(b di.ContainerBuilder){
		
        //register dependencies as a constructor method
        b.Provide(echo.New)
        
        //register dependencies with a factory method 
        b.Provide(func() config.Config {
            return config.Config{
                Database: config.Database{
                    Host: "localhost",
                    Port: 5432,
                },
                Server: config.Server{
                    Addr: ":8080",
                },
            }
        })
        
        //factory methods can have dependencies
        b.Provide(func(cfg config.Config) *repository.UserRepository {
            return repository.NewUserRepository(cfg.Database.Host, cfg.Database.Port)
        })
        
        //register an instance without a constructor method, singleton like pattern
        b.Provide(di.Instance(echo.New()))
		
		//create an alias for the echo dependency
        b.Provide(Alias[*echo.Echo, http.Handler]())
    })


    //as an example we use the Invoke method to inject the depencies into the callback function and execute it
    //In this example there is an echo dependency registered
    container.Invoke(func(h *echo.Echo, config config.Config) {
        h.Start(config.Server.Addr())
    })
```
