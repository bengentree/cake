package vsphere

const (
	uploadPort       string = "50000"
	commandPort      string = "50001"
	remoteExecutable string = "/tmp/cake"
	remoteConfig     string = "~/.cake.yaml"
	baseFolder       string = "cake"
	templatesFolder  string = "templates"
	workloadsFolder  string = "workloads"
	mgmtFolder       string = "mgmt"
	bootstrapFolder  string = "bootstrap"
	bootstrapVMName  string = "BootstrapVM"
	uploadFileCmd    string = "socat -u TCP-LISTEN:%s,fork CREATE:%s,group=root,perm=0755 & disown"
	runRemoteCmd     string = "socat TCP-LISTEN:%s,reuseaddr,fork EXEC:'/bin/bash',pty,setsid,setpgid,stderr,ctty & disown"
	runLocalCakeCmd  string = "%s deploy --local --deployment-type %s > /tmp/cake.out"
)
