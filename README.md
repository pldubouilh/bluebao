bluebao
======

![a](https://user-images.githubusercontent.com/760637/115114220-58dd4280-9f8e-11eb-9382-33a38c50bc91.png)

simple audio devices bluetooth manager, that lives on the tray - and supports easy multi-device usage through local network broadcasting.

upon startup it will show all audio devices paired. selecting a device will connect to it and select it as default audio output. selecting another device will disconnect the first one.

upon connecting a device, it will broadcast a message on the local network, so that other devices using bluebao can disconnect from the bluetooth device.

### usage
```
% ./bluebao --help
🥟 bluebao
A simple bluetooth audio devices manager, that supports local network broadcasting
to easily manage multiple devices on an bluetooth audio sink.

  -d    disable network feature
  -sp string
        server port (default "8829")
```

### build

depends on `bluetoothctl` at runtime and `gtk3 libappindicator3` for the build. cross distro builds are not so nicely performed because of libc dependency, but a binaries for latest ubuntu and arch are available on github.


