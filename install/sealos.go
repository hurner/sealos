package install

import (
	"fmt"
	"strings"

	"github.com/wonderivan/logger"
)

type CleanCluster interface {
	Check
	Clean
}

type JoinNodeAndMaster interface {
	Check
	Send
	Join
}

type Init interface {
	Check
	Send
	PreInit
	Join
	Print
}

type Install interface {
	Check
	Send
	Apply
}

var (
	JoinToken       string
	TokenCaCertHash string
	CertificateKey  string
)

//SealosInstaller is
type SealosInstaller struct {
	Hosts     []string
	Masters   []string
	Nodes     []string
	Network   string
	ApiServer string
}

type CommandType string

//command type
const InitMaster CommandType = "initMaster"
const JoinMaster CommandType = "joinMaster"
const JoinNode CommandType = "joinNode"

func (s *SealosInstaller) Command(version string, name CommandType) (cmd string) {
	cmds := make(map[CommandType]string)
	// Please convert your v1beta1 configuration files to v1beta2 using the
	// "kubeadm config migrate" command of kubeadm v1.15.x, 因此1.14 版本不支持双网卡.
	cmds = map[CommandType]string{
		InitMaster: `kubeadm init --config=/root/kubeadm-config.yaml --experimental-upload-certs` + vlogToStr(),
		JoinMaster: fmt.Sprintf("kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s --experimental-control-plane --certificate-key %s"+vlogToStr(), IpFormat(s.Masters[0]), JoinToken, TokenCaCertHash, CertificateKey),
		JoinNode:   fmt.Sprintf("kubeadm join %s:6443 --token %s --discovery-token-ca-cert-hash %s"+vlogToStr(), VIP, JoinToken, TokenCaCertHash),
	}
	//other version >= 1.15.x
	//todo
	if VersionToInt(version) >= 115 {
		cmds[InitMaster] = `kubeadm init --config=/root/kubeadm-config.yaml --upload-certs` + vlogToStr()
		cmds[JoinMaster] = "kubeadm join --config=/root/kubeadm-join-config.yaml " + vlogToStr()
		cmds[JoinNode] = "kubeadm join --config=/root/kubeadm-join-config.yaml " + vlogToStr()
	}

	// version >= 1.16.x support kubeadm init --skip-phases=addon/kube-proxy
	// version <= 115
	// kubectl -n kube-system delete ds kube-proxy
	// # Run on each node:
	// iptables-restore <(iptables-save | grep -v KUBE)
	if s.Network == "cilium" {
		if VersionToInt(version) >= 116 {
			cmds[InitMaster] = `kubeadm init --skip-phases=addon/kube-proxy --config=/root/kubeadm-config.yaml --upload-certs` + vlogToStr()
		} else {
			cmds[InitMaster] = `kubeadm init --config=/root/kubeadm-config.yaml --upload-certs` + vlogToStr()
		}
	}

	v, ok := cmds[name]
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[globals]fetch command error")
		}
	}()
	if !ok {
		panic(1)
	}
	return v
}

//decode output to join token  hash and key
func decodeOutput(output []byte) {
	s0 := string(output)
	logger.Debug("[globals]decodeOutput: %s", s0)
	slice := strings.Split(s0, "kubeadm join")
	slice1 := strings.Split(slice[1], "Please note")
	logger.Info("[globals]join command is: %s", slice1[0])
	decodeJoinCmd(slice1[0])
}

//  192.168.0.200:6443 --token 9vr73a.a8uxyaju799qwdjv --discovery-token-ca-cert-hash sha256:7c2e69131a36ae2a042a339b33381c6d0d43887e2de83720eff5359e26aec866 --experimental-control-plane --certificate-key f8902e114ef118304e561c3ecd4d0b543adc226b7a07f675f56564185ffe0c07
func decodeJoinCmd(cmd string) {
	logger.Debug("[globals]decodeJoinCmd: %s", cmd)
	stringSlice := strings.Split(cmd, " ")

	for i, r := range stringSlice {
		switch r {
		case "--token":
			JoinToken = stringSlice[i+1]
		case "--discovery-token-ca-cert-hash":
			TokenCaCertHash = stringSlice[i+1]
		case "--certificate-key":
			CertificateKey = stringSlice[i+1][:64]
		}
	}
}
