bluebao
======

![a](https://user-images.githubusercontent.com/760637/115114220-58dd4280-9f8e-11eb-9382-33a38c50bc91.png)

simple multi device bluetooth manager for local networks that lives in the systray - allows to connect a bluetooth device, disconnect other hosts from a device, etc...

a seed configuration is defined with the config below (assuming pairing is already done), the clients will then automatically disconnect/connect from the devices and UDP broadcast the information on the local network.

depends on `bluetoothctl` (at runtime) and `gtk3 libappindicator3` (for the build). cross builds are not so nicely performed because of libc dependency, but a dockerfile is provided to build against latest ubuntu.

### usage
```
% make

% ./bluebao --help
Usage of ./bluebao:
  -cp string
        client port (default "8830")
  -h string
        multicast address (default "192.168.0.255")
  -sp string
        server port (default "8829")
```

### todo
  - only run on wifi network XXX


