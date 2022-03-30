package ListenerSignalOS

import (

)

type Listener interface {
   start(parametr chan bool)
   stop(parametr chan bool)
   ListenerSignalOS(parametr chan bool)
}