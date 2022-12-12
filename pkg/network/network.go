package network

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"sort"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

const (
	DefaultNetworkRootDir = "/var/run/my-docker/network"
	DefaultNetworkDir     = DefaultNetworkRootDir + "/networks"
)

func init() {
	if err := util.EnsureDirectory(DefaultNetworkDir); err != nil {
		logrus.Warningf("faile to ensure network diectory %v: %v", DefaultNetworkDir, err)
	}
}

type Network struct {
	Name    string     `json:"name"`
	Driver  string     `json:"driver"`
	Subnet  *net.IPNet `json:"subnet"`
	Gateway *net.IPNet `json:"gateway"`
}

func (n *Network) String() string {
	body, _ := json.Marshal(n)
	return string(body)
}

func CreateNetwork(driver, subnet, name string) error {
	network, _ := readNetwork(name)
	if network != nil {
		logrus.Warningf("network %v already exists", name)
		return fmt.Errorf("network %v already exists", name)
	}

	if _, ok := drivers[driver]; !ok {
		logrus.Errorf("do not support driver %v", driver)
		return fmt.Errorf("do not support driver %v", driver)
	}

	var err error
	network, err = drivers[driver].CreateNetwork(name, subnet)
	if err != nil {
		logrus.Errorf("failed to create network (%v): %v", name, err)
		return err
	}
	logrus.Infof("network %v has been created successfully", network)

	if err = saveNetwork(network); err != nil {
		logrus.Errorf("failed to save network (%v): %v", name, err)
		return err
	}
	logrus.Infof("network %v has been saved successfully", network)
	return nil
}

func ListNetworks() error {
	files, err := ioutil.ReadDir(DefaultNetworkDir)
	if err != nil {
		logrus.Errorf("failed to read directory (%v): %v", DefaultNetworkDir, err)
		return err
	}

	networks := make([]*Network, len(files))
	var network *Network
	for i := range files {
		if files[i].IsDir() {
			continue
		}
		network, err = readNetwork(files[i].Name())
		if err != nil {
			logrus.Errorf("failed to read network (%v): %v", files[i].Name(), err)
			return err
		}
		networks[i] = network
	}
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "NAME\tSUBNET\tGATEWAY\tDRIVER\n")
	for _, n := range networks {
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", n.Name, n.Subnet, n.Gateway.IP, n.Driver)
	}
	if err = w.Flush(); err != nil {
		return fmt.Errorf("flush ps write err: %v", err)
	}
	return nil
}

func DeleteNetwork(name string) error {
	network, err := readNetwork(name)
	if err != nil {
		logrus.Errorf("failed to get network (%v): %v", name, err)
		return err
	}

	if _, ok := drivers[network.Driver]; !ok {
		logrus.Errorf("do not support driver %v", network.Driver)
		return fmt.Errorf("do not support driver %v", network.Driver)
	}

	if err = drivers[network.Driver].DeleteNetwork(network); err != nil {
		logrus.Errorf("failed to delete network (%v): %v", name, err)
		return err
	}

	if err = removeNetwork(name); err != nil {
		logrus.Errorf("failed to save network (%v): %v", name, err)
		return err
	}
	return nil
}

func readNetwork(networkName string) (*Network, error) {
	networkPath := generateNetworkPath(networkName)
	body, err := ioutil.ReadFile(networkPath)
	if err != nil {
		logrus.Errorf("failed to read %v: %v", networkPath, err)
		return nil, err
	}
	network := &Network{}
	if err = json.Unmarshal(body, network); err != nil {
		logrus.Errorf("failed to unmarshal %v: %v", string(body), err)
		return nil, err
	}
	return network, nil
}

func saveNetwork(network *Network) error {
	body, err := json.Marshal(network)
	if err != nil {
		logrus.Errorf("failed to marshal network (%+v): %v", network, err)
		return err
	}

	networkName := network.Name
	if err = util.EnsureDirectory(DefaultNetworkDir); err != nil {
		logrus.Errorf("failed to ensure network directory %v: %v", DefaultNetworkDir, err)
	}

	var file *os.File
	configPath := generateNetworkPath(networkName)
	file, err = os.Create(configPath)
	if err != nil {
		logrus.Errorf("failed to create file of network (%v): %v", configPath, err)
		return err
	}

	if _, err = file.Write(body); err != nil {
		logrus.Errorf("failed to write (%v) to network file (%v): %v", string(body), configPath, err)
		return err
	}
	return nil
}

func removeNetwork(networkName string) error {
	configPath := generateNetworkPath(networkName)
	if err := os.RemoveAll(configPath); err != nil {
		logrus.Errorf("failed to remove network config (%v): %v", configPath, err)
		return err
	}
	return nil
}

func generateNetworkPath(networkName string) string {
	return path.Join(DefaultNetworkDir, networkName)
}
