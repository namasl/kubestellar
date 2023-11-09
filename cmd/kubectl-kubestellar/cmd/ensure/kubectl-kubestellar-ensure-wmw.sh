#!/usr/bin/env bash
want_kube=true
wmw_name=""

while (( $# > 0 )); do
    case "$1" in
	(-h|--help)
	    echo "Usage: kubectl kubestellar ensure wmw (\$kubectl_flag | --with-kube boolean | -X)* wm_workspace_name"
	    exit 0;;
	(-X) set -o xtrace;;
	(--with-kube)
	    if (( $# >1 ))
	    then want_kube="$2"; shift
	    else echo "$0: missing with-kube value" >&2; exit 1
	    fi;;
	(--context*)
	    # TODO: support --context
	    echo "$0: --context flag not supported" >&2; exit 1;;
	(--*=*|-?=*)
	    kubectl_flags[${#kubectl_flags[*]}]="$1";;
	(--*|-?)
	    kubectl_flags[${#kubectl_flags[*]}]="$1";
	    if (( $# > 1 )); then 
		 kubectl_flags[${#kubectl_flags[*]}]="$2"
		 shift
	    fi;;
	(-*)
	    echo "$0: flag syntax error" >&2
	    exit 1;;
	(*)
	    if [ -z "$wmw_name" ]
	    then wmw_name="$1"
	    else echo "$0: too many positional arguments" >&2
		 exit 1
	    fi
    esac
    shift
done

if [ "$wmw_name" == "" ]; then
    echo "$0: workload management workspace name not specified" >&2
    exit 1
fi

case "$want_kube" in
    (true|false) ;;
    (*) echo "$0: with-kube should be true or false" >&2
	exit 1;;
esac

set -e

kubectl ws "${kubectl_flags[@]}" root

if kubectl "${kubectl_flags[@]}" get workspaces.tenancy.kcp.io "$wmw_name" &> /dev/null
then kubectl ws "${kubectl_flags[@]}" "$wmw_name"
else kubectl ws "${kubectl_flags[@]}" create "$wmw_name" --enter
fi

if ! kubectl "${kubectl_flags[@]}" get APIBinding bind-espw &> /dev/null; then
kubectl "${kubectl_flags[@]}" apply -f - <<EOF
apiVersion: apis.kcp.io/v1alpha1
kind: APIBinding
metadata:
  name: bind-espw
spec:
  reference:
    export:
      path: root:espw
      name: edge.kubestellar.io
EOF
fi

function bind_iff_wanted() { # usage: export_name
    export_name=$1
    binding_name=bind-$export_name
    if [ "$want_kube" == true ] && ! kubectl "${kubectl_flags[@]}" get APIBinding ${binding_name} &> /dev/null; then
kubectl "${kubectl_flags[@]}" apply -f - <<EOF
apiVersion: apis.kcp.io/v1alpha1
kind: APIBinding
metadata:
  name: ${binding_name}
spec:
  reference:
    export:
      path: root:compute
      name: ${export_name}
EOF
elif [ "$want_kube" == false ] && kubectl "${kubectl_flags[@]}" get APIBinding ${binding_name} &> /dev/null; then
     kubectl "${kubectl_flags[@]}" delete APIBinding ${binding_name}
fi
}

bind_iff_wanted kubernetes
bind_iff_wanted apiregistration.k8s.io
bind_iff_wanted apps
bind_iff_wanted autoscaling
bind_iff_wanted batch
bind_iff_wanted core.k8s.io
bind_iff_wanted cluster-core.k8s.io
bind_iff_wanted discovery.k8s.io
bind_iff_wanted flowcontrol.apiserver.k8s.io
bind_iff_wanted networking.k8s.io
bind_iff_wanted cluster-networking.k8s.io
bind_iff_wanted node.k8s.io
bind_iff_wanted policy
bind_iff_wanted scheduling.k8s.io
bind_iff_wanted storage.k8s.io
bind_iff_wanted cluster-storage.k8s.io