Howto
=====

Make climc as an ansible inventory source.

1. Edit /etc/ansible/hosts

```
#!/bin/bash

export OS_USERNAME=sysadmin
export OS_PASSWORD=<password>
export OS_PROJECT_NAME=system
export OS_DOMAIN_NAME=Default
export OS_AUTH_URL=http://10.168.200.246:5000/v3
export OS_REGION_NAME=YunionHQ

/opt/yunion/bin/climc ansible-hosts --port 22 --user yunion --private-key $HOME/priv-key --user-become root $@
```


2. Make /etc/ansible/hosts executable

```sh
chmod +x /etc/ansible/hosts
```

3. Try

```sh
ansible-inventory --list
ansible-inventory --host <hostname>
```
