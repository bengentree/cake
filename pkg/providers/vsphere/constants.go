package vsphere

const (
	uploadPort       = "50000"
	commandPort      = "50001"
	remoteExecutable = "/tmp/cake"
	remoteConfig     = "~/.cake.yaml"
	baseFolder       = "cake"
	templatesFolder  = "templates"
	workloadsFolder  = "workloads"
	mgmtFolder       = "mgmt"
	bootstrapFolder  = "bootstrap"
	bootstrapVMName  = "BootstrapVM"
	uploadFileCmd    = "socat -u TCP-LISTEN:%s,fork CREATE:%s,group=root,perm=0755 & disown"
	runRemoteCmd     = "socat TCP-LISTEN:%s,reuseaddr,fork EXEC:'/bin/bash',pty,setsid,setpgid,stderr,ctty & disown"
	runLocalCakeCmd  = "%s deploy --local --deployment-type %s > /tmp/cake.out"
)
