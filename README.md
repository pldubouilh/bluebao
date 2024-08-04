bluebao
======

![a](https://github.com/user-attachments/assets/fe42fa37-b43b-47c7-bf31-8bc7d270adfb)

simple audio devices bluetooth manager, that lives on the tray.

### features
 + connect to bluetooth audio devices (disconnecting any other connected audio device)
 + select default bluetooth profile (a2dp, hsp, etc)
 + a client/server mechanism to disconect other bluebao clients from a device if a bluebao instance connects it

### usage
```
% ./bluebao --help
🥟 bluebao
A simple bluetooth audio devices manager to easily manage multiple devices.

  -e    enable network feature
  -sp string
        server port (default "8829")
```

### build
depends on `bluetoothctl` at runtime and `gtk3 libappindicator3` for the build. cross distro builds are not so nicely performed because of libc dependency, but a binaries for latest ubuntu and arch are available on github.


