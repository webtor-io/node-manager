# node-manager
Manages nodes in hybrid k8s self-hosted cluster

## Supported providers
- [x] Contabo
- [ ] Hetzner Robot (dedicated)

## Supported commands
- [x] Heal - reboots all nodes in NotReady state
- [x] Reboot - reboots node by its name in k8s cluster

## Setting up

First of all set label `provider={{your_provider}}` for your nodes like this:

```
kubectl get no | awk '{print $1}' | xargs -I{} kubectl label nodes {} provider=contabo
```

Then setup credentials for you provider with env variables:

```
export CONTABO_CLIENT_ID={{PUT_SOMETHING_THERE}} \
export CONTABO_CLIENT_SECRET={{PUT_SOMETHING_THERE}} \
export CONTABO_API_USER={{PUT_SOMETHING_THERE}} \
export CONTABO_API_PASSWORD={{PUT_SOMETHING_THERE}}
```

## Usage
```
./node-manager help
NAME:
   node-manager - Manages nodes

USAGE:
   node-manager [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   heal, h    reboots all NotReady nodes
   reboot, r  reboots node
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```
