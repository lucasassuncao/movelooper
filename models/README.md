<!-- gomarkdoc:embed:start -->

<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# models

```go
import "movelooper/models"
```

## Index

- [type MediaConfig](<#MediaConfig>)
- [type Movelooper](<#Movelooper>)
- [type PersistentFlags](<#PersistentFlags>)


<a name="MediaConfig"></a>
## type [MediaConfig](<https://github.com/lucasassuncao/movelooper/blob/main/models/movelooper.go#L15-L20>)



```go
type MediaConfig struct {
    CategoryName string   `mapstructure:"name"`
    Extensions   []string `mapstructure:"extensions"`
    Source       string   `mapstructure:"source"`
    Destination  string   `mapstructure:"destination"`
}
```

<a name="Movelooper"></a>
## type [Movelooper](<https://github.com/lucasassuncao/movelooper/blob/main/models/movelooper.go#L8-L13>)



```go
type Movelooper struct {
    Logger      *pterm.Logger
    Viper       *viper.Viper
    Flags       *PersistentFlags
    MediaConfig []*MediaConfig
}
```

<a name="PersistentFlags"></a>
## type [PersistentFlags](<https://github.com/lucasassuncao/movelooper/blob/main/models/flags.go#L3-L7>)



```go
type PersistentFlags struct {
    Output     *string
    LogLevel   *string
    ShowCaller *bool
}
```

Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)


<!-- gomarkdoc:embed:end -->