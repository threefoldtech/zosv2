module github.com/threefoldtech/zos

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/aristanetworks/goarista v0.0.0-20201012165903-2cb20defcd66 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/containerd/cgroups v0.0.0-20200327175542-b44481373989
	github.com/containerd/containerd v1.4.0-beta.1.0.20200615192441-ae2f3fdfd1a4
	github.com/containerd/typeurl v0.0.0-20190911142611-5eb25027c9fd
	github.com/containernetworking/cni v0.7.2-0.20190807151350-8c6c47d1c7fc
	github.com/containernetworking/plugins v0.8.4
	github.com/deckarep/golang-set v1.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ethereum/go-ethereum v1.9.22 // indirect
	github.com/firecracker-microvm/firecracker-go-sdk v0.19.1-0.20200110212531-741fc8cb0f2e
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gizak/termui/v3 v3.1.0
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/leesmet/go-substrate-rpc-client v2.0.3-0.20201014152134-2262065fe495+incompatible
	github.com/opencontainers/runtime-spec v1.0.1
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/rs/cors v1.7.0 // indirect
	github.com/rs/zerolog v1.19.0
	github.com/shirou/gopsutil v2.20.5+incompatible
	github.com/stretchr/testify v1.6.1
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/threefoldtech/tfexplorer v0.4.1-0.20201006092040-c7f28e901ea5
	github.com/threefoldtech/zbus v0.1.3
	github.com/urfave/cli v1.22.4
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df
	github.com/whs/nacl-sealed-box v0.0.0-20180930164530-92b9ba845d8d
	github.com/yggdrasil-network/yggdrasil-go v0.3.15-0.20200526002434-ed3bf5ef0736
	golang.org/x/crypto v0.0.0-20201012173705-84dcc777aaee
	golang.org/x/net v0.0.0-20201010224723-4f7140c49acb
	golang.org/x/sys v0.0.0-20201014080544-cc95f250f6bc
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20191219145116-fa6499c8e75f
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
