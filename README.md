bluebao
======

![a](https://github.com/user-attachments/assets/fe42fa37-b43b-47c7-bf31-8bc7d270adfb)

simple bluetooth audio devices manager, that lives on the tray.

### features
 + connect to bluetooth audio devices (disconnecting any other connected audio device)
 + select default bluetooth profile (a2dp, hsp, etc)
 + an optional client/server mechanism to disconect other bluebao clients on the local network from a device when an instance requires a connection

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
depends on `bluetoothctl` at runtime and `gtk3 libappindicator3` for the build. cross distro builds are not so nicely performed because of libc dependency, but a binaries for latest ubuntu, arch, and ubuntu-lts are available on github.

### nits
bluebao calls `bluetoothctl` directly. I initially wanted to use dbus, but the endpoint requires root, and I'd rather keep bluebao root-less.


