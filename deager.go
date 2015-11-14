package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/fsouza/go-dockerclient"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	usage := `Eager Docker Client

Usage:
  deager [options] (start|stop|gui|run)
  deager -h | --help
  deager --version

start:    Spins up the EAGER docker container
stop:     Stop/remove the EAGER container
gui:      Connect to container and start eager GUI
run:      Run eagercli within --data directory

Options:
  --gatk <path>	     Path to the gtak file (jar/tar.bz2) [default: ~/gatk/]
                     It has to be provided by the user, since the license prohibits packaging.
  --data <path>      Directory to use as /data/ directory within eager (default: ~/data)
  --image <str>      Name of the eager image [default: peltzer/eager]
  --container <str>  Name of the container spun up (default: eager_$USER)
  -h --help          Show this screen.
  --version          Show version.`
	arguments, _ := docopt.Parse(usage, nil, true, "Eager Docker Client 0.9", false)
	if arguments["--gatk"] == nil {
		arguments["--gatk"] = fmt.Sprintf("%s/gatk/", os.Getenv("HOME"))
	}
	if arguments["--data"] == nil {
		arguments["--data"] = fmt.Sprintf("%s/data/", os.Getenv("HOME"))
	}
	if arguments["--container"] == nil {
		arguments["--container"] = fmt.Sprintf("eager_%s", os.Getenv("USER"))
	}
	if arguments["--image"] == nil {
		arguments["--image"] = "peltzer/eager"
	}
	//fmt.Println(arguments)
	client := gimmeDocker()
	if arguments["start"].(bool) {
		// TODO: Create a struct for arguments?
		err := startEager(client, arguments["--image"].(string), arguments["--container"].(string),
			arguments["--data"].(string), arguments["--gatk"].(string))
		check(err)
	}
	if arguments["stop"].(bool) {
		err := stopEager(client, arguments["--container"].(string))
		check(err)
	}
	if arguments["gui"].(bool) {
		re1, err := regexp.Compile(`(\d+\.\d+\.\d+\.\d+)`)
		result := re1.FindStringSubmatch(os.Getenv("DOCKER_HOST"))
		host := result[0]
		ssh_key, err := writeSshKey()
		check(err)
		cmd := exec.Command("ssh", "-p", "2222", "-i", ssh_key, "-l", "eager", host, "eager")
		err = cmd.Start()
		check(err)
		err = cmd.Wait()
		check(err)
	}
	if arguments["run"].(bool) {
		err := runEagercli()
		check(err)
	}
}

func runEagercli() error {
	config := &ssh.ClientConfig{
		User: "eager",
		Auth: []ssh.AuthMethod{
			ssh.Password("eager"),
		},
	}
	host, _ := getDockerHostIP()
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:2222", host), config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	// TODO: Create StdoutPipe
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = &b
	//cmd := "for x in {1..5};do echo $x;sleep 1;done "
	cmd := "eagercli /data/"
	if err := session.Run(cmd); err != nil {
		fmt.Println("eagercli was not successful!")
		fmt.Println(b.String())
		return err
	}
	fmt.Println(b.String())
	return nil
}

func writeSshKey() (name string, err error) {
	content := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA00T3YMiYqt2yqgK6PoKR2YfLqiwQ98B3yocs3KWANAbi889H
Blsnz/RoG8aUbhN2B7lrzV8Wjv/io53HRlhIKYXGh99QWeV9exvu3dI3rIW1Xdt9
p7RnQDf3T+mnr34SaH0OMLip+dqMJQw3nv/SZPJtXeP6F0M85mSqg6U2ZJmq966f
VRFOx1i2lJQuWEwlyIUTS/FYSg05qA8G8Tjs4MLjwuJnWVT8YTVpB/f7LxftfB6g
1tsBtvIj9im4iINl4g1XY6ocxy/V52IQ8p/9s6LeDQcD5HCvbTAasZrgrdnPZx59
mbhhq1PctZjDASaDE/lLDNXK/CgWLoB4xBQvGQIDAQABAoIBAG6lWPWsOSCLmW2m
ngns8hu+HfESwRQwDczY/KrWVo1o6eWMsgLnLLOhqgCaANShhphHCOl3GmZsJzNP
h7UUuT5d3Hr+fqOGKDCYkYJE/XlyUWlFccqqFcUxSmnk0jh7y4JDtHHZ1NORHQKu
Ilc4XeUWfibFJg6W3UdAg3kMxq7qQ/iHKqGzl5D2BTWIvreX5rBkkGyQyp0KiFel
DPoq/hZpwJ8JemOxgxekMXU6jhoXLm+PC9HJrEi0pghl2828Opc3VUEUQLGcBLFi
zrEjGLEwMITnhzzCRv2cZCN/u+2PU/IWZFiOGeLSn5/r8e0ftFC4ITg2d6s6wkBj
l7TbKQECgYEA9nyYD17kJqVidBb3hN/NSFBXXWNSAqMTRM4bg7sEXwdUMYf9JtOR
y6ZeVczKS2wsl5NIYWDkJw2ZwaDNYVmMWVYFQT0BMkZgv6q3EJJJm70AgJVihwlM
NnwY7ZtFueQhU+vG2XD62dnKKNa6egmrnWhJXKlAWz+OOWI3K9F9n6ECgYEA22xo
nsSRYBj4UxEgq0m9oHLlYP33mQi6JUkxUSQNJ/ZjQSjxKQEgsUEovPR00uTS4exn
7PKyjj0CnsE998MAkE54IBzRdmHGqfVHRcVo7vFiYV4TAe3vJzB6WroZ4AyArHWD
w0i6DTNO7QV8BQqIL12QIN2uwG7fSmSVb9IKPHkCgYAITuDNO9SS3OY5pYCIUQbZ
ViPruOpNvnNq0UuqIAagsV2MIdpNkboLVDs/xxxWeHn0TfmVlq96BYJWPXZOvrb1
V+nrbgP5Ttf5/eYXv+aNQkyfCOn+RTj1aS9p6t7pyh+5dWwJbj52U1n2EG7OqD7J
mndGkUnjCXxgwMe9SV1joQKBgHsegDGd8Ehwml3ZvW//J3SxI33h4x0uZWxoflCe
Hveua5DzTSYJ6PMssZQcwrRXCvETulic8Y2YNDEqEwBDnbxbG1JBeVKomFVjOIOw
uilgrigeJiIuBMQDkpP32m759PVP1wgrdaHUiVO7gRQ/DZ0uLaITYWu+inHusF8X
BwFZAoGBAMH5nPtn1vc5kzb/u1YCdtLuM0nVRXDDZix4AsVzP6cTqr9jrCQNcCJf
oWCIMnZWl5oP8CglJT6/c1t/5bAz95wnB3SZpBJn0qxcn2d9uVACOO7H3VpjdbiA
9A4YNOGQlG8Ov5Xr2ve+9flT5g8St8PA8YWtUk3w67vZ6YyDYyPX
-----END RSA PRIVATE KEY-----`)
	fname, err := ioutil.TempFile("/tmp", "eager_ssh_key_")
	check(err)
	err = ioutil.WriteFile(fname.Name(), content, 0600)
	check(err)
	return fname.Name(), nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func checkForGatk(path string) (gatkpath string, err error) {
	files, _ := ioutil.ReadDir(path)
	gpath := ""
	for _, f := range files {
		match, _ := regexp.MatchString("genomeanalysistk", strings.ToLower(f.Name()))
		if match {
			gpath = path
		}
	}
	if gpath == "" {
		errstr := fmt.Sprintf("Could not find 'GenomeAnalysisTK.*' in '%s'", path)
		return "", errors.New(errstr)
	}
	return gpath, nil
}

func gimmeDocker() (cli *docker.Client) {
	endpoint := os.Getenv("DOCKER_HOST")
	if os.Getenv("DOCKER_TLS_VERIFY") == "1" {
		path := os.Getenv("DOCKER_CERT_PATH")
		ca := fmt.Sprintf("%s/ca.pem", path)
		cert := fmt.Sprintf("%s/cert.pem", path)
		key := fmt.Sprintf("%s/key.pem", path)
		client, _ := docker.NewTLSClient(endpoint, cert, key, ca)
		return client
	} else {
		client, _ := docker.NewClient(endpoint)
		return client
	}
}

func startEager(client *docker.Client, image string, containerName string, data string, gatk string) error {
	exposedCadvPort := map[docker.Port]struct{}{"22/tcp": {}}

	createContConf := docker.Config{
		ExposedPorts: exposedCadvPort,
		Image:        image,
	}

	portBindings := map[docker.Port][]docker.PortBinding{
		"22/tcp": {{HostIP: "0.0.0.0", HostPort: "2222"}}}

	gatkPath, _ := checkForGatk(gatk)
	gatkBind := fmt.Sprintf("%s:/opt/gatk/", gatkPath)
	dataBind := fmt.Sprintf("%s:/data/", data)
	createContHostConfig := docker.HostConfig{
		// Figure out where gatk is located and add it to the bind-mounts
		Binds:           []string{gatkBind, dataBind},
		PortBindings:    portBindings,
		PublishAllPorts: false,
		Privileged:      true,
	}

	createContOps := docker.CreateContainerOptions{
		Name:       containerName,
		Config:     &createContConf,
		HostConfig: &createContHostConfig,
	}

	cont, err := client.CreateContainer(createContOps)
	if err != nil {
		fmt.Printf("create error = %s\n", err)
		return err
	}
	err = client.StartContainer(cont.ID, nil)
	if err != nil {
		fmt.Printf("start error = %s\n", err)
		return err
	}
	return nil
}

func stopEager(client *docker.Client, containerName string) error {
	err := client.StopContainer(containerName, 5)
	if err != nil {
		fmt.Printf("stop error = %s\n", err)
		return err
	}
	rmOpts := docker.RemoveContainerOptions{
		ID:            containerName,
		RemoveVolumes: true,
		Force:         true,
	}
	err = client.RemoveContainer(rmOpts)
	if err != nil {
		fmt.Printf("remove error = %s\n", err)
		return err
	}
	return nil
}

func getDockerHostIP() (host string, err error) {
	re1, err := regexp.Compile(`(\d+\.\d+\.\d+\.\d+)`)
	result := re1.FindStringSubmatch(os.Getenv("DOCKER_HOST"))
	ip := result[0]
	return ip, nil
}
