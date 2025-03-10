---
short_name: commands
---

This PoC includes two sorts of commands for users to use. Most are
executables designed to be accessed in the usual ways: by appearing in
a directory on the user's `$PATH` or by the user writing a pathname
for the executable on the command line. The one exception is [the
bootstrap script](#bootstrap), which is designed to be fetched from
github and fed directly into `bash`.

This document describes both commands that a platform administrator
uses and commands that a platform user uses.

## Deployment style

There normal way of deploying the space framework, a space provider,
and the central KubeStellar controllers is as workload in a Kubernetes
(possibly OpenShift) cluster.  Contributors may instead run the Space
Manager, possibly space provider (dependening on the provider), and
KubeStellar central controllers as bare processes on a machine of
supported OS and ISA. Even in this case a Kubernetes cluster is
required, at least to hold the Space Management API objects; this role
is called the Space Management API Space (SMAS). The KubeFlex space
provider must be deployed in a cluster, which is called the KubeFlex
hosting cluster. That cluster can also play the SMAS role.

### In-cluster

Some commands take an optional flag `--in-cluster` that is about the
networking relationship between (a) that running command and (b) the
apiserver(s) for the space(s) that the command accesses. This flag can
be used when both (a) and (b) are in the same Kubernetes cluster. This
flag can also be used when both (a) and (b) are _not_ in a Kubernetes
cluster but rather are bare processes on the same machine. Omitting
this flag means that the running command will use the more general way
of addressing the apiserver(s), which is functional in more situations
but possibly less efficient.

### OS namespace consistency

For some of the commands, all of them must always be invoked in the
context of the same filesystem, working directory, and process
space. This is a point-in-time limitation due to the way kube-bind
konnectors are launched. The relevant commands are as follows.

- `kubectl kubestellar ensure inventory-space`
- `kubectl kubestellar ensure wmw`
- `kubestellar` (which is also constrained because `start` and `stop`
  are implemented by ordinary commands, no remote agent is involved)
- `kubestellar-kube-bind`

## Bare process deployment

### kcp process control

KubeStellar has some commands to support situations in which some
clients of the kcp server need to connect to it by opening a
connection to a DNS domain name rather than an IP address that the
server can bind its socket to.  The key idea is using the
`--tls-sni-cert-key` command line flag of `kcp start` to configure the
server to respond with a bespoke server certificate in TLS handshakes
in which the client addresses the server by a given domain name.

These commands are used separately from starting the kcp server and
are designed so that they can be used multiple times if there are
multiple sets of clients that need to use a distinct domain name.
Starting the kcp server is not described here, beyond the particular
consideration needed for the `--tls-sni-cert-key` flag, because it is
otherwise ordinary.

#### kubestellar-ensure-kcp-server-creds

##### kubestellar-ensure-kcp-server-creds pre-reqs

The `kubestellar-ensure-kcp-server-creds` command requires that
Easy-RSA is installed.  As outlined in
https://easy-rsa.readthedocs.io/en/latest/#obtaining-and-using-easy-rsa,
this involves selecting a release archive from [the list on
GitHub](https://github.com/OpenVPN/easy-rsa/releases), unpacking it,
and adding the EasyRSA directory to your `$PATH`; `easyrsa` is a bash
script, so you do not need to worry about building or fetching a
binary specific to your OS or computer architecture.

Easy-RSA uses [OpenSSL](https://www.openssl.org/), so you will need
that installed too.

##### kubestellar-ensure-kcp-server-creds usage

This command is given exactly one thing on the command line, a DNS
domain name.  This command creates --- or re-uses if it finds already
existing --- a private key and public X.509 certificate for the kcp
server to use.  The certificate has exactly one
SubjectAlternativeName, which is of the DNS form and specifies the
given domain name.  For example: `kubestellar-ensure-kcp-server-creds
foo.bar` creates a certificate with one SAN, commonly rendered as
`DNS:foo.bar`.

This command uses a Public Key Infrastructure (PKI) and Certificate
Authority (CA) implemented by
[easy-rsa](https://github.com/OpenVPN/easy-rsa), rooted at the
subdirectory `pki` of the current working directory.  This command
will create the PKI if it does not already exist, and will initialize
the CA if that has not already been done.  The CA's public certificate
appears at the usual place for easy-rsa: `pki/ca.crt`.

This command prints some stuff --- mostly progress remarks from
easy-rsa --- to stderr and prints one line of results to stdout.  The
`bash` shell will parse that one line as three words.  Each is an
absolute pathname of one certificate or key.  The three are as
follows.

1. The CA certificate for the client to use in verifying the server.
2. The X.509 certificate for the server to put in its ServerHello in a
   TLS handshake in which the ClientHello has a Server Name Indicator
   (SNI) that matches the given domain name.
3. The private key corresponding to that server certificate.

Following is an example of invoking this command and examining its
results.

```console
bash-5.2$ eval pieces=($(kubestellar-ensure-kcp-server-creds yep.yep))
Re-using PKI at /Users/mspreitz/go/src/github.com/kubestellar/kubestellar/pki
Re-using CA at /Users/mspreitz/go/src/github.com/kubestellar/kubestellar/pki/private/ca.key
Accepting existing credentials

bash-5.2$ echo ${pieces[0]}
/Users/mspreitz/go/src/github.com/kubestellar/kubestellar/pki/ca.crt

bash-5.2$ echo ${pieces[1]}
/Users/mspreitz/go/src/github.com/kubestellar/kubestellar/pki/issued/kcp-DNS-yep.yep.crt

bash-5.2$ echo ${pieces[2]}
/Users/mspreitz/go/src/github.com/kubestellar/kubestellar/pki/private/kcp-DNS-yep.yep.key
```

Following is an example of using those results in launching the kcp
server.  The `--tls-sni-cert-key` flag can appear multiple times on
the command line, configuring the server to respond in a different way
to each of multiple different SNIs.

```console
bash-5.2$ kcp start --tls-sni-cert-key ${pieces[1]},${pieces[2]} &> /tmp/kcp.log &
[1] 66974
```

#### wait-and-switch-domain

This command is for using after the kcp server has been launched.
Since the `kcp start` command really means `kcp run`, all usage of
that server has to be done by concurrent processes.  The
`wait-and-switch-domain` command bundles two things: waiting for the
kcp server to start handling kubectl requests, and making an alternate
kubeconfig file for a set of clients to use.  This command is pointed
at an existing kubeconfig file and reads it but does not write it; the
alternate config file is written (its directory must already exist and
be writable).  This command takes exactly six command line positional
arguments, as follows.

1. Pathname (absolute or relative) of the input kubeconfig.
2. Pathname (absolute or relative) of the output kubeconfig.
3. Name of the kubeconfig "context" that identifies what to replace.
4. Domain name to put in the replacement server URLs.
5. Port number to put in the replacement server URLs.
6. Pathname (absolute or relative) of a file holding the CA
   certificate to put in the alternate kubeconfig file.

Creation of the alternate kubeconfig file starts by looking in the
input kubeconfig file for the "context" with the given name, to find
the name of a "cluster".  The server URL of that cluster is examined,
and its `protocol://host:port` prefix is extracted.  The alternate
kubeconfig will differ from the input kubeconfig in the contents of
the cluster objects whose server URLs start with that same prefix.
There will be the following two differences.

1. In the server URL's `protocol://host:port` prefix, the host will be
   replaced by the given domain name and the port will be replaced by
   the given port.
2. The cluster will be given a `certificate-authority-data` that holds
   the contents (base64 encoded, as usual) of the given CA certificate
   file.

Following is an example of using this command and examining the
results.  The context and port number chosen work for the kubeconfig
file that `kcp start` (kcp release v0.11.0) creates by default.

```console
bash-5.2$ wait-and-switch-domain .kcp/admin.kubeconfig test.yaml root yep.yep 6443 ${pieces[0]}

bash-5.2$ diff -w .kcp/admin.kubeconfig test.yaml
4,5c4,5
<     certificate-authority-data: LS0...LQo=
<     server: https://192.168.something.something:6443
---
>       certificate-authority-data: LS0...LQo=
>       server: https://yep.yep:6443
8,9c8,9
<     certificate-authority-data: LS0...LQo=
<     server: https://192.168.something.something:6443/clusters/root
---
>       certificate-authority-data: LS0...LQo=
>       server: https://yep.yep:6443/clusters/root
```

Following is an example of using the alternate kubeconfig file, in a
context where the domain name "yep.yep" resolves to an IP address of
the network namespace in which the kcp server is running.

```console
bash-5.2$ KUBECONFIG=.kcp-yep.yep/admin.kubeconfig kubectl ws .
Current workspace is "root".
```

Because this command reads the given kubeconfig file, it is important
to invoke this command while nothing is concurrently writing it and
while the caller reliably knows the name of a kubeconfig context that
identifies what to replace.

#### switch-domain

This command is the second part of `wait-and-switch-domain`: the part
of creating the alternate kubeconfig file.  It has the same inputs and
outputs and concurrency considerations.

### KubeStellar process control

The `kubestellar` command has three subcommands, one to finish setup
and two for process control.

Except for the `stop` subcommand, this command has the following
pre-requisites.

- The space manager is running, there is a space provider named
  "default", and that is the one to use.

- When not given any command-line flags, `kubectl` will access the
  Space Management API Space (SMAS).

This command is one of those requiring [OS namespace
consistency](#os-namespace-consistency).

The usage synopsis is as follows.

``` { .bash .no-copy }
kubestellar [flags] subcommand [flags]
```

This command accepts the following command line flags, which can
appear before and/or after the subcommand.  The `--log-folder` flag is
only meaningful for the `start` subcommand. The `--in-cluster` flag is
not meaningful for the `stop` subcommand. The `--ensure-imw` and 
`--ensure-wmw` flags are only meaningful for the `start` or `init` subcommands.

- `-V` or `--verbose`: calls for more verbose output.  This is a
  binary choice, not a matter of degree.
- `-X`: turns on echoing of script lines
- `--log-folder $pathname`: says where to put the logs from the
  controllers.  Will be `mkdir -p` if absent.  Defaults to
  `${PWD}/kubestellar-logs`.
- `--in-cluster`: as [usual](#in-cluster).
- `--ensure-imw`: provide a comma separated list of pathnames for inventory workspaces, _e.g._ "root:imw1,root:imw2". Defaults to "root:imw1". To prevent the creation of any inventory workspace, then pass "".
- `--ensure-wmw`: provide a comma separated list of pathnames for workload management workspaces, _e.g._ "root:wmw1,root:imw2". Defaults to "root:wmw1". To prevent the creation of any workload management workspace, then pass "".
- `-h` or `--help`: print a brief usage message and terminate.

#### Kubestellar init

This subcommand is used after installation to finish setup and does
the following five things.

1. Ensure that the KubeStellar Core Space (KCS, formerly called edge
   service provider workspace (ESPW)) exists and has the required
   contents.

2. Ensure the existence of zero, one, or more inventory spaces
   depending on the value of `--ensure-imw` flag. Default is one
   inventory space named "imw1".

3. Ensure the existence of zero, one, or more workload description
   spaces (WDS, formerly workload management workspaces (WMW))
   depending on the value of `--ensure-wmw` flag. Default is one WDS
   named "wmw1". The WDSes have CRDs for the Kubernetes APIS for
   management of containerized workloads.

#### KubeStellar start

This subcommand is used after installation or process stops.

This subcommand stops any running kubestellar controllers and then
starts them all.  It also does the same things as `kubestellar init`.

#### KubeStellar stop

This subcommand undoes the primary function of `kubestellar start`,
stopping any running KubeStellar controllers.  It does _not_ tear down
the ESPW.

## Deployment into a Kubernetes cluster

These commands administer a deployment of the central components ---
the space provider server, PKI, and the central KubeStellar components 
--- in a Kubernetes cluster that will be referred to as "the hosting 
cluster".  These commands are framed as "kubectl plugins" and thus need 
to be explicitly or implicitly given a kubeconfig file for the hosting
cluster.

You need a Kubernetes cluster with an Ingress controller deployed and
configured in a way that does _not_ terminate TLS connections (this
abstinence is often called "SSL passthrough"). An OpenShift cluster
would be one qualifying thing. Another would be an ordinary Kubernetes
cluster with the [nginx Ingress
controller](https://docs.nginx.com/nginx-ingress-controller/) deployed
and configured appropriately. Please note that [special
considerations](https://kind.sigs.k8s.io/docs/user/ingress/) apply
when deploying an ingress controller in `kind`. See [a fully worked
example with kind and
nginx](../environments/dev-env/#hosting-kubestellar-in-a-kind-cluster). You
will need to know the port number at which the Ingress controller is
listening for HTTPS connections.

**IF** your Kubernetes cluster has any worker nodes --- real or
virtual --- with the x86_64 instruction set, they need to support the
extended instruction set known as "x64-64-v2". If using hardware
bought in the last 10 years, you can assume that is true. If using
emulation, you need to make sure that your emulator is emulating that
extended instruction set --- some emulators do not do this by
default. See [QEMU configuration
recommendations](https://www.qemu.org/docs/master/system/i386/cpu.html),
for example.

You will need a domain name that, on each of your clients, resolves to
an IP address that gets to the Ingress controller's listening socket.

### Deploy to cluster

Deployment is done with a "kubectl plugin" that is invoked as `kubectl
kubestellar deploy` and creates a [Helm
"release"](https://helm.sh/docs/intro/using_helm/#three-big-concepts)
in the hosting cluster. As such, it relies on explicit (on the command
line) or implicit (in environment variables and/or `~/.kube/config`)
configuration needed to execute Helm commands. The following flags can
appear on the command line, in any order.

- `--openshift $bool`, saying whether the hosting cluster is an
  OpenShift cluster.  If so then a Route will be created to the kcp
  server; otherwise, an Ingress object will direct incoming TLS
  connections to the kcp server.  The default is `false`.
- `--external-endpoint $domain_name:$port`, saying how the kcp server
  will be reached from outside the cluster.  The given domain name
  must be something that the external clients will resolve to an IP
  address where the cluster's Ingress controller or OpenShift router
  will be listening, and the given port must be the corresponding TCP
  port number.  For a plain Kubernetes cluster, this must be
  specified.  For an OpenShift cluster this may be omitted, in which
  case the command will (a) assume that the external port number is
  443 and (b) extract the external hostname from the Route object
  after it is created and updated by OpenShift.  FYI, that external
  hostname will start with a string derived from the Route in the
  chart (currently "kubestellar-route-kubestellar") and continue with
  "." and then the ingress domain name for the cluster.
- a command line flag for the `helm upgrade` command. This includes
  the usual control over namespace: you can set it on the command
  line, otherwise the namespace that is current in your kubeconfig
  applies.
- `-X` turns on debug echoing of all the commands in the script that
  implements this command.
- `-h` prints a brief usage message and terminates with success.

For example, to deploy to a plain Kubernetes cluster whose Ingress
controller can be reached at
`my-long-application-name.my-region.some.cloud.com:1234`, you would
issue the following command.

```shell
kubectl kubestellar deploy --external-endpoint my-long-application-name.my-region.some.cloud.com:1234
```

The Helm chart takes care of setting up the KubeStellar Core,
accomplishing the same thing as the [kubestellar
start](#kubestellar-start) command above.

### Fetch a log from a KubeStellar runtime container

{%
   include-markdown "../../../../core-helm-chart/README.md"
   start="<!--check-log-start-->"
   end="<!--check-log-end-->"
%}

### Remove deployment to a Kubernetes cluster

The deployment of kcp and KubeStellar as a Kubernetes workload is done
by making a "release" (this is a [technical term in
Helm](https://helm.sh/docs/intro/cheatsheet/#basic-interpretationscontext)
whose meaning might surprise you) of a Helm chart.  This release is
named "kubestellar".  To undo the deployment, just use the Helm
command to delete that release.

```shell
helm delete kubestellar
```

**NOTE**: this is a detail that might change in the future.

## KubeStellar platform user commands

The remainder of the commands in this document are for users rather
than administrators of the service that KubeStellar provides.

## KubeStellar-release

This command just echoes the [semantic version](https://semver.org/)
of the release used.  This command is only available in archives built
for a release.  Following is an example usage.

```shell
kubestellar-release
```
``` { .bash .no-copy }
v0.2.3-preview
```

## Kubestellar-version

This executable prints information about itself captured at build
time.  If built by `make` then this is information conveyed by the
Makefile; otherwise it is [the Kubernetes
defaults](https://github.com/kubernetes/client-go/blob/master/pkg/version/base.go).

It will either print one requested property or a JSON object
containing many.

```shell
kubestellar-version help
```
``` { .bash .no-copy }
Invalid component requested: "help"
Usage: kubestellar-version [buildDate|gitCommit|gitTreeState|platform]
```

```shell
kubestellar-version buildDate
```
``` { .bash .no-copy }
2023-05-19T02:54:01Z
```

```shell
kubestellar-version gitCommit
```
``` { .bash .no-copy }
1747254b
```

```shell
kubestellar-version
```
``` { .bash .no-copy }
{"major":"1","minor":"24","gitVersion":"v1.24.3+kcp-v0.2.1-20-g1747254b880cb7","gitCommit":"1747254b","gitTreeState":"dirty","buildDate":"2023-05-19T02:54:01Z","goVersion":"go1.19.9","compiler":"gc","platform":"darwin/amd64"}
```

## Creating an Inventory Space

This command will create an inventory space (IS) of a given name if it
does not already exist and ensure that it is properly prepared for
use.

This command is one of those requiring [OS namespace
consistency](#os-namespace-consistency).

Preparing for use includes using kube-bind to import the definitions
of `SyncTarget` and `Location`. If that has not already been done then
it will be done by invoking
[kubestellar-kube-bind](#kubestellar-kube-bind), with the attendant
side-effects to the filesystem (possibly creating
`${PWD}/konnector-${space_name}/` and/or a temporary kubeconfig file
in `$PWD`) and constraint on working directory (always the same for a
given WDS).

This command is given a kube client configuration for access to the
Space Management API Space (SMAS), via the usual `kubectl` flags and
`$KUBECONFIG`.

The usage synopsis for this command is as follows.

```shell
kubectl kubestellar ensure inventory-space flag... inventory_space_name
```

This command accepts the following flags. They can be intermixed with
the positional argument.

- `-h` or `--help`: print a brief usage message and terminate
  successfully.
- `-X`: turn on debug echoing of the commands inside the script that
  implements this command.
- a `kubectl` flag (but not `-o` or `--output`).
- `--in-cluster`: as [usual](#in-cluster).
- `--output-kubeconfig $file_pathname`: requests that an in-cluster
  kubeconfig for the space be output to the given file pathname.

This script works in idempotent style, doing whatever work remains to
be done.

## Creating SyncTarget/Location pairs

In this PoC, the interface between infrastructure and workload
management is inventory API objects.  Specifically, for each workload
execution cluster there is a unique pair of SyncTarget and Location
objects in a so-called inventory space.  These kinds of objects were
originally defined in kcp TMC, and now there is a copy of those
definitions (CRDs) in KubeStellar.  It is the definitions in
KubeStellar that should be referenced.  Those are in the Kubernetes
API group `edge.kubestellar.io`, and they are exported from the
[KubeStellar Core Space
(KCS)](../../../../Getting-Started/user-guide/)) (the space named
`espw`).

The following command helps with making that SyncTarget and Location
pair. This command operates on an existing inventory space. This
command is given a kube client config for that space via the usual
`kubectl` command line flags and `$KUBECONFIG`.

The usage synopsis is as follows.

```shell
kubectl kubestellar ensure location flag... objname labelname=labelvalue...
```

Here `objname` is the name for the SyncTarget object and also the name
for the Location object.  This command ensures that these objects exist
and have at least the given labels.

The flags can also appear anywhere later on the command line.

This command accepts the following flags.

- `-h` or `--help`: print a brief usage message and terminate successfully.
- `-X`: turn on debug echoing of the commands inside the script that
  implements this command.
- a `kubectl` flag (but not `-o` or `--output`).

This command does not depend on the action of any of the KubeStellar
controllers but does require that the KubeStellar Core Space (KCS) has
been set up.

An example usage follows.

```shell
KUBECONFIG=imw1.kubeconfig kubectl kubestellar ensure location demo1 foo=bar the-word=the-bird
```
``` { .bash .no-copy }
synctarget.workload.kcp.io/demo1 created
location.scheduling.kcp.io/demo1 created
synctarget.workload.kcp.io/demo1 labeled
location.scheduling.kcp.io/demo1 labeled
synctarget.workload.kcp.io/demo1 labeled
location.scheduling.kcp.io/demo1 labeled
```

The above example shows using this script to create a SyncTarget and a
Location named `demo1` with labels `foo=bar` and `the-word=the-bird`.
This was equivalent to the following commands.

```shell
KUBECONFIG=imw1.kubeconfig kubectl create -f -<<EOF
apiVersion: workload.kcp.io/v1alpha1
kind: SyncTarget
metadata:
  name: demo1
  labels:
    id: demo1
    foo: bar
    the-word: the-bird
---
apiVersion: scheduling.kcp.io/v1alpha1
kind: Location
metadata:
  name: demo1
  labels:
    foo: bar
    the-word: the-bird
spec:
  resource: {group: workload.kcp.io, version: v1alpha1, resource: synctargets}
  instanceSelector:
    matchLabels: {"id":"demo1"}
EOF
```

Note that the extraction of space kubeconfig file can be accomplished using the 
`kubectl kubestellar space get-kubeconfig` command.

This command operates in idempotent style, making whatever changes (if
any) are needed to move from the current state to the desired state.

## Removing SyncTarget/Location pairs

THIS SCRIPT IS BROKEN

The following script undoes whatever remains from a corresponding
usage of `kubectl kubestellar ensure location`.  It has all the same
command line syntax and semantics except that the
`labelname=labelvalue` pairs do not appear.

This command does not depend on the action of any of the KubeStellar controllers.

The following session demonstrates usage, including idempotency.

```shell
KUBECONFIG=$imw_space_config kubectl kubestellar remove location demo1
```
``` { .bash .no-copy }
synctarget.workload.kcp.io "demo1" deleted
location.scheduling.kcp.io "demo1" deleted
```

```shell
KUBECONFIG=$imw_space_config kubectl kubestellar remove location demo1
```

## Getting a kubeconfig of a given space

This command fetches a kubeconfig for super-user access to a given
space into a given filename. The kubeconfig will be good for one year
since the creation of the space unless it was configured unusually.

Each space has two such kubeconfig files. One kubeconfig is known as
the "in-cluster" kubeconfig and is for accessing the space from within
the same Kubernetes cluster as the space's apiserver(s) (assuming they
_are_ in a cluster). The other is known as the "external" kubeconfig
and is for accessing the space from outside the space's host cluster
(assuming there is one). Clients inside that cluster generally can use
the external kubeconfig successfully but it is less efficient. The
precise truth of the matter depends on the hostname configured for
external access at deployment time.

This command is given a client configuration for accessing the Space
Management API Space (SMAS), via the usual ways for configuring
`kubectl`.

The usage synopsis is as follows.

```shell
kubectl kubestellar space get-kubeconfig flag... $space_name $output_pathname
```

This command accepts the following flags.

- `-h` or `--help`: print usage and terminate with success.
- `-X`: turn on debug echoing of the script as it executes.
- a `kubectl` flag (but not `-o` or `--output`).
- `--in-cluster`: indicates that the kubeconfig is for use
  [in-cluster](#in-cluster).

Following is an example usage.

```shell
kubectl kubestellar space get-kubeconfig imw1 imw1.kubeconfig
```

## Syncer preparation and installation

The syncer runs in each edge cluster and also talks to the
corresponding mailbox space.  In order for it to be able to do
that, there is some work to do in the mailbox space to create a
ServiceAccount for the syncer to authenticate as and create RBAC
objects to give the syncer the privileges that it needs.  The
following script does those things and also outputs YAML to be used to
install the syncer in the edge cluster.

This script is given a kube client config for accessing the SMAS via
the usual `kubectl` command line flags and `$KUBECONFIG`.

The usage synopsis is as follows.

```shell
kubectl kubestellar prep-for-syncer flag... synctarget_name
```

Here `synctarget_name` is the name of the `SyncTarget` object, in the
relevant IMW, corresponding to the relevant edge cluster.

The flags can also appear anywhere later on the command line.

This command accepts the following flags.

- `-h` or `--help`: print a brief usage message and terminate successfully.
- `-X`: turn on debug echoing of commands inside the script that
  implements this command.
- a `kubectl` flaag (but not `-o` or `--output`).
- `--in-cluster`: as [usual](#in-cluster).
- `--imw space_name`: specifies which space holds the relevant
  SyncTarget object.
- `--espw space_name`: specifies where to find the edge service
  provider space.
- `--syncer-image image_ref`: specifies the container image that runs
  the syncer.  The default is `quay.io/kubestellar/syncer:{{ config.ks_tag }}`.
- `-o output_pathname`: specifies where to write the YAML definitions
  of the API objects to create in the edge cluster in order to deploy
  the syncer there.  The default is `synctarget_name +
  "-syncer.yaml"`. **NOTE** this is differnt from the usual meaning of
  `-o` for `kubectl`.
- `-s`: exceptionally low info output.

This command will only succeed if the mailbox controller has created
and conditioned the mailbox space for the given SyncTarget.  This
command will wait for 10 to 70 seconds for that to happen.

An example usage follows.

```shell
kubectl kubestellar prep-for-syncer --imw imw-1 demo1
```
``` { .bash .no-copy }
Current space is "4yqm57kx0m6mn76c-mb-406c54d1-64ce-4fdc-99b3-cef9c4fc5010" 
Creating service account "kubestellar-syncer-demo1-28at01r3"
Creating cluster role "kubestellar-syncer-demo1-28at01r3" to give service account "kubestellar-syncer-demo1-28at01r3"

 1. write and sync access to the synctarget "kubestellar-syncer-demo1-28at01r3"
 2. write access to apiresourceimports.

Creating or updating cluster role binding "kubestellar-syncer-demo1-28at01r3" to bind service account "kubestellar-syncer-demo1-28at01r3" to cluster role "kubestellar-syncer-demo1-28at01r3".

Wrote WEC manifest to demo1-syncer.yaml for namespace "kubestellar-syncer-demo1-28at01r3". Use

  KUBECONFIG=<space-execution-cluster-config> kubectl apply -f "demo1-syncer.yaml"

to apply it. Use

  KUBECONFIG=<space-execution-cluster-config> kubectl get deployment -n "kubestellar-syncer-demo1-28at01r3" kubestellar-syncer-demo1-28at01r3

to verify the syncer pod is running.
```

Once that script has run, the YAML for the objects to create in the
edge cluster is in your chosen output file.  The default for the
output file is the name of the SyncTarget object with "-syncer.yaml"
appended.

Create those objects with a command like the following; adjust as
needed to configure `kubectl` to modify the edge cluster and read your
chosen output file.

```shell
KUBECONFIG=$demo1_kubeconfig kubectl apply -f demo1-syncer.yaml
```

You can preserve this file for future use if you want to remove the
syncer later.

## Syncer removal

The previous section shows the two steps involved in establishing a
syncer: one modifies the core and one adds stuff into the WEC. If for
any reason your core gets deleted (e.g., deliberate replacement) and
you want to undo the addition of the syncer into the WEC, you can use
the YAML file generated above with `kubectl delete` instead of
`kubectl apply` (of course using the right kubeconfig and context
therein).

A syncer that is configured to work with an extinct core is harmless
except that its pod will be doing some futile work (failing to make
requests on the core and logging error messages).

If you need to manually remove the syncer from your WEC, following is
the list of things that went into the WEC.

- A namespace named
  `kubestellar-syncer-${SyncTarget_Name}-${UID}`. The UID is a random
  ID that is uniquely generated each time you run prep-for-syncer.
- A ServiceAccount (with associated token Secret) in that namespace.
- Another Secret in that namespace.
- A Deployment in that namespace
- A ClusterRole with the same name as the namespace.
- A ClusterRoleBinding with the same name as the namespace.

## Workload execution cluster on-boarding

The following command is a combination of `kubectl kubestellar
ensure-location` and `kubectl kubestellar prep-for-syncer`, and takes
the union of their command line flags and arguments with the following
exceptions/modifications.

- The `-o` flag is not accepted.
- The `kubectl` flags and `$KUBECONFIG` say how to access the Space
  Management API Space (SMAS), not the inventory space.

An example usage follows.

```shell
kubectl kubestellar prep-for-cluster --imw imw-1 demo2 key1=val1
```
``` { .bash .no-copy }
synctarget.workload.kcp.io/demo2 created
location.scheduling.kcp.io/demo2 created
synctarget.workload.kcp.io/demo2 labeled
location.scheduling.kcp.io/demo2 labeled
Current space is "1cpf1cd4ydy13vo1-mb-3c354acd-ed86-45bb-a60d-cee8e59973f7" 
Creating service account "kubestellar-syncer-demo2-15nq4e94"
Creating cluster role "kubestellar-syncer-demo2-15nq4e94" to give service account "kubestellar-syncer-demo2-15nq4e94"

 1. write and sync access to the synctarget "kubestellar-syncer-demo2-15nq4e94"
 2. write access to apiresourceimports.

Creating or updating cluster role binding "kubestellar-syncer-demo2-15nq4e94" to bind service account "kubestellar-syncer-demo2-15nq4e94" to cluster role "kubestellar-syncer-demo2-15nq4e94".

Wrote WEC manifest to demo2-syncer.yaml for namespace "kubestellar-syncer-demo2-15nq4e94". Use

  KUBECONFIG=<space-execution-cluster-config> kubectl apply -f "demo2-syncer.yaml"

to apply it. Use

  KUBECONFIG=<space-execution-cluster-config> kubectl get deployment -n "kubestellar-syncer-demo2-15nq4e94" kubestellar-syncer-demo2-15nq4e94

to verify the syncer pod is running.
```

## Creating a Workload Description Space

This command will create a WDS of a given name if it does not already
exist and ensure that it is properly prepared for use.

This command is one of those requiring [OS namespace
consistency](#os-namespace-consistency).

Preparing for use includes using kube-bind to import the relevant
parts of the KubeStellar API. If that has not already been done then
it will be done by invoking
[kubestellar-kube-bind](#kubestellar-kube-bind), with the attendant
side-effects to the filesystem (possibly creating
`${PWD}/konnector-${space_name}/` and/or a temporary kubeconfig file
in `$PWD`) and constraint on working directory (always the same for a
given WDS).

Preparing for use also includes ensuring the CRDs for the Kubernetes
APIs for management of containerized workloads and present iff
desired.

This command is given a kube client configuration for access to the
Space Management API Space (SMAS), via the usual `kubectl` flags and
`$KUBECONFIG`.

The usage synopsis for this command is as follows.

```shell
kubectl kubestellar ensure wmw flag... wm_space_name
```

This command accepts the following flags. They can be intermixed with
the positional argument.

- `-h` or `--help`: print a brief usage message and terminate
  successfully.
- `-X`: turn on debug echoing of the commands inside the script that
  implements this command.
- `--in-cluster`: as [usual](#in-cluster).
- a `kubectl` flag (but not `-o` or `--output`).
- `--output-kubeconfig $file_pathname`: requests that an in-cluster
  kubeconfig for the WDS be output to the given file pathname.
- `--with-kube boolean`: specifies whether or not the WMW should
  include CRDs for the supported subset of the Kubernetes API
  for management of containerized workloads. See [the categorization
  in the design](../outline/#data-objects) for details.

This script works in idempotent style, doing whatever work remains to
be done.

The following session shows some example usages, including
demonstration of idempotency and changing whether the kube CRDs
are included.

```shell
kubectl kubestellar ensure wmw example-wmw
```
``` { .bash .no-copy }
Error from server (NotFound): spaces.space.kubestellar.io "example-wmw" not found
space.space.kubestellar.io/example-wmw created
Waiting for space example-wmw to get into the Ready phase.
Space example-wmw created.
SECRET_NAME=external-example-wmw
SECRET_NAMESPACE=spaceprovider-default
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
starting kube-bind konnector for space example-wmw in background, logs writing to /home/ubuntu/kubestellar/kubestellar-kube-bind-logs/kube-bind-konnector-example-wmw.log
Storing konnector kubeconfigs in /home/fred/kubestellar-konnectors/231212-010203-edgeplacements-1234
binding edgeplacements for example-wmw
cluster namespace is kube-bind-tmflg
Current workspace is "root:espw".
namespace kubestellar already exists
configmap/kbmap-fcf8hzj7mlx156cm created
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
kube-bind konnector for space example-wmw already running
Storing konnector kubeconfigs in /home/fred/kubestellar-konnectors/231212-010213-customizers-1245
binding customizers for example-wmw
cluster namespace is kube-bind-tmflg
Current workspace is "root:espw".
namespace kubestellar already exists
configmap/kbmap-fcf8hzj7mlx156cm configured
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
kube-bind konnector for space example-wmw already running
Storing konnector kubeconfigs in /home/fred/kubestellar-konnectors/231212-010233-singleplacementslices-1256
binding singleplacementslices for example-wmw
cluster namespace is kube-bind-tmflg
Current workspace is "root:espw".
namespace kubestellar already exists
configmap/kbmap-fcf8hzj7mlx156cm configured
customresourcedefinition.apiextensions.k8s.io/apiservices.apiregistration.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/persistentvolumes.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/flowschemas.flowcontrol.apiserver.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/prioritylevelconfigurations.flowcontrol.apiserver.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/ingressclasses.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/runtimeclasses.node.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/priorityclasses.scheduling.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/csidrivers.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/csinodes.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/storageclasses.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/volumeattachments.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/daemonsets.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/deployments.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/replicasets.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/statefulsets.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/horizontalpodautoscalers.autoscaling serverside-applied
customresourcedefinition.apiextensions.k8s.io/cronjobs.batch serverside-applied
customresourcedefinition.apiextensions.k8s.io/jobs.batch serverside-applied
customresourcedefinition.apiextensions.k8s.io/endpoints.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/persistentvolumeclaims.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/pods.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/podtemplates.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/replicationcontrollers.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/services.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/endpointslices.discovery.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/ingresses.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/networkpolicies.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/poddisruptionbudgets.policy serverside-applied
customresourcedefinition.apiextensions.k8s.io/csistoragecapacities.storage.k8s.io serverside-applied
```

```shell
kubectl kubestellar ensure wmw example-wmw
```
``` { .bash .no-copy }
SECRET_NAME=external-example-wmw
SECRET_NAMESPACE=spaceprovider-default
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
CRD for edgeplacements already in place
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
CRD for customizers already in place
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
CRD for singleplacementslices already in place
customresourcedefinition.apiextensions.k8s.io/apiservices.apiregistration.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/persistentvolumes.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/flowschemas.flowcontrol.apiserver.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/prioritylevelconfigurations.flowcontrol.apiserver.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/ingressclasses.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/runtimeclasses.node.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/priorityclasses.scheduling.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/csidrivers.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/csinodes.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/storageclasses.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/volumeattachments.storage.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/daemonsets.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/deployments.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/replicasets.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/statefulsets.apps serverside-applied
customresourcedefinition.apiextensions.k8s.io/horizontalpodautoscalers.autoscaling serverside-applied
customresourcedefinition.apiextensions.k8s.io/cronjobs.batch serverside-applied
customresourcedefinition.apiextensions.k8s.io/jobs.batch serverside-applied
customresourcedefinition.apiextensions.k8s.io/endpoints.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/persistentvolumeclaims.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/pods.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/podtemplates.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/replicationcontrollers.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/services.core serverside-applied
customresourcedefinition.apiextensions.k8s.io/endpointslices.discovery.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/ingresses.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/networkpolicies.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/poddisruptionbudgets.policy serverside-applied
customresourcedefinition.apiextensions.k8s.io/csistoragecapacities.storage.k8s.io serverside-applied
```

```shell
IN_CLUSTER=false kubectl kubestellar ensure wmw example-wmw --with-kube false
```
``` { .bash .no-copy }
SECRET_NAME=external-example-wmw
SECRET_NAMESPACE=spaceprovider-default
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
CRD for edgeplacements already in place
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
CRD for customizers already in place
Current workspace is "root".
kcp workspace example-wmw's ID is fcf8hzj7mlx156cm
Current workspace is "root:example-wmw".
CRD for singleplacementslices already in place
customresourcedefinition.apiextensions.k8s.io "apiservices.apiregistration.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "persistentvolumes.core" deleted
customresourcedefinition.apiextensions.k8s.io "flowschemas.flowcontrol.apiserver.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "prioritylevelconfigurations.flowcontrol.apiserver.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "ingressclasses.networking.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "runtimeclasses.node.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "priorityclasses.scheduling.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "csidrivers.storage.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "csinodes.storage.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "storageclasses.storage.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "volumeattachments.storage.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "daemonsets.apps" deleted
customresourcedefinition.apiextensions.k8s.io "deployments.apps" deleted
customresourcedefinition.apiextensions.k8s.io "replicasets.apps" deleted
customresourcedefinition.apiextensions.k8s.io "statefulsets.apps" deleted
customresourcedefinition.apiextensions.k8s.io "horizontalpodautoscalers.autoscaling" deleted
customresourcedefinition.apiextensions.k8s.io "cronjobs.batch" deleted
customresourcedefinition.apiextensions.k8s.io "jobs.batch" deleted
customresourcedefinition.apiextensions.k8s.io "endpoints.core" deleted
customresourcedefinition.apiextensions.k8s.io "persistentvolumeclaims.core" deleted
customresourcedefinition.apiextensions.k8s.io "pods.core" deleted
customresourcedefinition.apiextensions.k8s.io "podtemplates.core" deleted
customresourcedefinition.apiextensions.k8s.io "replicationcontrollers.core" deleted
customresourcedefinition.apiextensions.k8s.io "services.core" deleted
customresourcedefinition.apiextensions.k8s.io "endpointslices.discovery.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "ingresses.networking.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "networkpolicies.networking.k8s.io" deleted
customresourcedefinition.apiextensions.k8s.io "poddisruptionbudgets.policy" deleted
customresourcedefinition.apiextensions.k8s.io "csistoragecapacities.storage.k8s.io" deleted
```

## Removing a Workload Description Space

THIS SCRIPT IS BROKEN

Deleting a WDS (formerly WMW) can be done by simply deleting its
`space` object from the parent.

```shell
KUBECONFIG=$SM_CONFIG kubectl delete space -n spaceprovider-default example-wmw
```
``` { .bash .no-copy }
workspace.tenancy.kcp.io "example-wmw" deleted
```

Alternatively, you can use the following command line whose design
completes the square here.  Invoke it when the current workspace is
the parent of the workload management workspace to delete.

```shell
kubectl kubestellar remove wmw -h
```
``` { .bash .no-copy }
Usage: kubectl kubestellar remove wmw [-X] kubectl_flag... wm_workspace_name
```

```shell
kubectl kubestellar remove wmw demo1
```
``` { .bash .no-copy }
Current workspace is "root".
workspace.tenancy.kcp.io "demo1" deleted
```

```shell
kubectl kubestellar remove wmw demo1
```
``` { .bash .no-copy }
Current workspace is "root".
```

## kubestellar-list-syncing-objects

**NOTE**: This command works directly with the kcp server, it has not
  yet been converted to go through the space abstraction layer.

The `kubestellar-list-syncing-objects` command will list or watch one
kind of objects in the mailbox spaces. These are full (not summarized)
workload objects in the intermediate place between WDS' and WEC',
participating in downsync or upsync. The user of this command is not
very exposed to the mailbox spaces themselves; the command is directed
at the whole of this intermediate place and lists all of the requested
kind of objects there. The kind of objects to list/watch is given by
command line flags. In general the user has to also give the
"resource" that is roughly equivalent to the "kind", but that can be
defaulted in the easy case.

The output is suitable for piping to `jq` or `yq`. In the JSON case,
the output is one object per line (not pretty-printed). The default is
to output YAML.

This command will either do a one-shot listing or an ongoing
list+watch. In the latter case each object is extended with a field
named `Action` having a value of either `add`, `update`, or `delete`.

This command is given two Kubernetes client configurations.  One,
called "all", is for reading the chosen objects from all workspaces.
The other, called "parent", is for reading the mailbox Workspace
objects from their parent Workspace.

Following are the command line flags beyond the baseline golang flags.

```shell
      --api-group string                 API group of objects to watch (default is Kubernetes core group)
      --api-kind string                  kind of objects to watch
      --api-resource string              API resource (lowercase plural) of objects to watch (defaults to lowercase(kind)+'s')
      --api-version string               API version (just version, no group) of objects to watch (default "v1")
      --json                             indicates whether to output as lines of JSON rather than YAML
      --watch                            indicates whether to inform rather than just list
...
      --all-cluster string               The name of the kubeconfig cluster to use for access to the chosen objects in all clusters
      --all-context string               The name of the kubeconfig context to use for access to the chosen objects in all clusters (default "system:admin")
      --all-kubeconfig string            Path to the kubeconfig file to use for access to the chosen objects in all clusters
      --all-user string                  The name of the kubeconfig user to use for access to the chosen objects in all clusters
...
      --parent-cluster string              The name of the kubeconfig cluster to use for access to the parent of mailbox workspaces
      --parent-context string            The name of the kubeconfig context to use for access to the parent of mailbox workspaces (default "root")
      --parent-kubeconfig string           Path to the kubeconfig file to use for access to the parent of mailbox workspaces
      --parent-user string                 The name of the kubeconfig user to use for access to the parent of mailbox workspaces
```

Following is an example of its usage; the ellipses show where this
document omits many lines for brevity.

```console
$ kubestellar-list-syncing-objects --api-group apps --api-kind ReplicaSet
---
apiVersion: apps/v1
kind: ReplicaSet
metadata:
...
status:
  availableReplicas: 1
  fullyLabeledReplicas: 1
  observedGeneration: 1
  readyReplicas: 1
  replicas: 1

---
apiVersion: apps/v1
kind: ReplicaSet
metadata:
...
status:
  availableReplicas: 1
  fullyLabeledReplicas: 1
  observedGeneration: 1
  readyReplicas: 1
  replicas: 1
```

## Bootstrap

This is a combination of some installation and setup steps, for use in
[the
QuickStart](../../../Getting-Started/quickstart/).

The script can be read directly from
{{ config.repo_raw_url }}/{{ config.ks_branch }}/bootstrap/bootstrap-kubestellar.sh
and does the following things.

1. Downloads and installs kcp executables if they are not already
   evident on `$PATH`.
2. Downloads and installs kubestellar executables if they are not
   already evident on `$PATH`.
3. Ensures that kcp and KubeStellar are deployed (i.e., their
   processes are running and their initial configurations have been
   established) either as bare processes or as workload in a
   pre-existing Kubernetes cluster.

This script accepts the following command line flags; all are optional.  The `--os`, `--arch`, and `--bind-address` flags are only useful when deploying as bare processes.  The deployment will be into a Kubernetes cluster if either `--external-endpoint` or `--openshift true` is given.

- `--kubestellar-version $version`: specifies the release of
  KubeStellar to use.  When using a specific version, include the
  leading "v".  The default is the latest regular release, and the
  value "latest" means the same thing.
- `--kcp-version $version`: specifies the kcp release to use.  The
  default is the one that works with the chosen release of
  KubeStellar.
- `--openshift $bool`: specifies whether to the hosting cluster is an
  OpenShift cluster. The default value is `false`.
- `--endpoint-address $domain_name:$port`: specifies where an Ingress
  controller or OpenShift router is listening for incoming TLS
  connections from external (to the hosting cluster) clients of kcp
  and KubeStellar.
- `--os $OS`: specifies the operating system to use in selecting the
  executables to download and install.  Choices are `linux` and
  `darwin`.  Auto-detected if omitted.
- `--arch $IAS`: specifies the instruction set architecture to use in
  selecting the executables to download and install.  Choices are
  `amd64` and `arm64`.  Auto-detected if omitted.
- `--bind-address $IPADDR`: directs that the kcp server (a) write that
  address for itself in the kubeconfig file that it constructs and (b)
  listens only at that address.  The default is to pick one of the
  host's non-loopback addresses to write into the kubeconfig file and
  not bind a listening address.
- `--ensure-folder $install_parent_dir`: specifies the parent folder
  for downloads.  Will be `mkdir -p`.  The default is the current
  working directory.  The download of kcp, if any, will go in
  `$install_parent_dir/kcp`.  The download of KubeStellar will go in
  `$install_parent_dir/kubestellar`.
- `--host-ns $namespace_in_hosting_cluster`: specifies the namespace
  in the hosting cluster where the core will be deployed. Defaults to
  "kubestellar".
- `-V` or `--verbose`: increases the verbosity of output.  This is a
  binary thing, not a matter of degree.
- `-X`: makes the script `set -x` internally, for debugging.
- `-h` or `--help`: print brief usage message and exit.

Here "install" means only to (a) unpack the distribution archives into
the relevant places under `$install_parent_dir` and (b) enhance the
`PATH`, and `KUBECONFIG` in the case of kcp, environment variables in
the shell running the script.  Of course, if you run the script in a
sub-shell then those environment effects terminate with that
sub-shell; this script also prints out messages showing how to update
the environment in another shell.

## Internal commands

The following commands are used by other commands but not expected to
be invoked by users.

## kubestellar-kube-bind

This command invokes `kubectl bind` to bind one resource into a space.
Also, this command will launch a kube-bind `konnector` process if
there is not already one running locally that was launched by an
earlier invocation of this command.

This command is one of those requiring [OS namespace
consistency](#os-namespace-consistency).

This command is given a kube client config for accessing the SMAS, via
the usual `kubectl` flags and `$KUBECONFIG`.

Usage synposis is as follows.

```shell
kubestellar-kube-bind flag... space_name resource
```

This command accepts the following flags.

- `-h` or `--help`: prints usage and terminates successfully.
- `-X`: turn on echoing of script internals for debugging.
- a `kubectl` flag (but not `-o` or `--output`).
- `--in-cluster`: as [usual](#in-cluster).

If this command launches a `konnector` then this command will `mkdir
-p ${PWD}/konnector-$space_name` to hold the konnector's kubeconfig
input (named `kubeconfig`) and log file (named `log`).

Even if this invocation does not launch a konnector, this command will
dump some contents of `{$PWD}/konnector-${space_name}/log` when
reporting certain errors.

This command may also create a temporary kubeconfig file in `$PWD`.

## Ensure the existence of a space

This command will create a space if it does not already exist, and can
optionally output a kubeconfig for super-user access to that space.

This command is given a client configuration for accessing the Space
Management API Space (SMAS), via the usual ways for configuring
`kubectl`.

Synopsis:

```shell
kubectl kubestellar space ensure flag... space_name
```

This command accepts the following flags. The space name, output
kubeconfig, and SMAS kubeconfig and context must be supplied.

- `-h` or `--help`
- `-X`
- A `kubectl` flag (but not `-o` or `--output`).
- `--output-kubeconfig $pathname`
- `--in-cluster`: relevant iff requesting a kubeconfig for the space

