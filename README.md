bluebao
======

![a](https://user-images.githubusercontent.com/760637/115114220-58dd4280-9f8e-11eb-9382-33a38c50bc91.png)

simple multi device bluetooth manager for local networks that lives in the systray.

upon startup it will show all audio devices paired. selecting a device will connect to it and select it as default audio output. selecting another device will disconnect the first one.

upon connecting a device, it will broadcast a message on the local network, so that other devices (using bluebao) can disconnect from the device.

depends on `bluetoothctl` (at runtime) and `gtk3 libappindicator3` (for the build). cross distro builds are not so nicely performed because of libc dependency, but a binaries are available on github.

### usage
```
% make

% ./bluebao --help
Usage of ./bluebao:
  -cp string
        client port (default "8830")
  -sp string
        server port (default "8829")
```


