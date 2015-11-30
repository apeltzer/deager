package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/fsouza/go-dockerclient"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"io"
	"os"
	"log"
	"os/user"
	"os/exec"
	"regexp"
	"strings"
)

var (
    Trace   *log.Logger
    Info    *log.Logger
    Warning *log.Logger
    Error   *log.Logger
)

func Init(
    traceHandle io.Writer,
    infoHandle io.Writer,
    warningHandle io.Writer,
    errorHandle io.Writer) {

    Trace = log.New(traceHandle,
        "TRACE: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Info = log.New(infoHandle,
        "INFO: ",
        log.Ldate|log.Ltime)

    Warning = log.New(warningHandle,
        "WARNING: ",
        log.Ldate|log.Ltime)

    Error = log.New(errorHandle,
        "ERROR: ",
        log.Ldate|log.Ltime)
}


func main() {
	Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
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
  --image <str>      Name of the eager image [default: apeltzer/eager]
  --container <str>  Name of the container spun up (default: eager_$USER)
  --uid              Use docker-client UID/GID for eager user within container.
                     This will cope with user rights. (depends on bindmount; boot2docker, local docker deamon...)
  -h --help          Show this screen.
  --version          Show version.`
	arguments, _ := docopt.Parse(usage, nil, true, "Eager Docker Client 0.9", false)
	if arguments["--gatk"] == nil {
		arguments["--gatk"] = fmt.Sprintf("%s/gatk/", os.Getenv("HOME"))
	}
	if _, err := os.Stat(arguments["--gatk"].(string)); os.IsNotExist(err) {
		Error.Println("The gatk directory does not exist: ", arguments["--gatk"])
		os.Exit(1)
	}
	if arguments["--data"] == nil {
		arguments["--data"] = fmt.Sprintf("%s/data/", os.Getenv("HOME"))
	}
	if _, err := os.Stat(arguments["--data"].(string)); os.IsNotExist(err) {
		Error.Println("The data directory does not exist: ", arguments["--data"])
		os.Exit(1)
	}
	if arguments["--container"] == nil {
		arguments["--container"] = fmt.Sprintf("eager_%s", os.Getenv("USER"))
	}
	if arguments["--image"] == nil {
		arguments["--image"] = "peltzer/eager"
	}
	
	if os.Getenv("DOCKER_HOST") == "" {
		Error.Println("Please check your docker environment, DOCKER_HOST is not set.")
		Error.Println("Does the docker CLI work? >> docker ps")
		os.Exit(1)
	}
	//fmt.Println(arguments)
	client := gimmeDocker()
	if arguments["start"].(bool) {
		// TODO: Create a struct for arguments?
		err := startEager(client, arguments["--image"].(string), arguments["--container"].(string),
			arguments["--data"].(string), arguments["--gatk"].(string), arguments["--uid"].(bool))
		if err != nil {
			err_msg := fmt.Sprintln(err)
	        match, _ := regexp.MatchString(".*no such host", err_msg)
			if match {
				Error.Println("Please check your docker environment, DOCKER_HOST is not set correctly: ", os.Getenv("DOCKER_HOST"))
				Error.Println("Does the docker CLI work? >> docker ps")
			} else {
				Error.Println(err)
			}
			os.Exit(1)
		}
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
		cmd := exec.Command("ssh", "-X", "-p", "2222", "-i", ssh_key, "-l", "eager", host, "eager")
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
MIIEpAIBAAKCAQEA87IbB2l1MrVVfFAhKJRoAKGzmo6l/zIpq5phpY01fXA2TtLN
FU+DKcPYbgJUqFNGlOkBqZqizMSYorSH/YHYvBo3YpbfsAZf4sRsBo4xzBhW4g6f
0Kt4cidaYROTndr4T0syV2KbGNXp4sVJ5wCti5030cF59m/L18GFQGXLXy8FcqlH
srTkK9+jdLsd9oQYaqsV+0c8o40aUff0qgYPa6ARg4cUgLiyLRwl6inuQ/pDItyk
chQ5Iycfiwh4yB2/rpsGpWzbPtVRfwrTthaM+P2L4Kk1CXHJdVIDIkrM80hNujdo
Ox/roT6QTodnsZNs5njNfo9CuzmEGEJ+mQb3DwIDAQABAoIBAQC1/enflCs5LmDk
ELdipcoxxpDpuORQ+/ZQuF96EkXDIvz7ysPryVCr7R2Bsm3ksyQ/6u8Z6WjxQVS4
FdiFQuZIO8/m6cOtomUTZhtCngikYfzon4FMhfHSVn9Rhhw0xCWymfbDedlYJ9Ce
UTYKtN/mJwhbtoDNwNnbjCNmX18M+gynyxPJyVt4ksmvktdWMDG9XzE2Tw1XfiMt
53K9BCLaxcVGTEEjMkrTYPOXgdm6L7AlMkFZGeek+yZarz3fSdRYAhUhgFd9GXQ3
B+rFAc3xbEiBQmn+H5eOWCIi61QTTK9CnzUjEtnvNWOnggNzkCS0H9KEMRhEA91x
ePBbHzUpAoGBAPoFXZZrbZc70NLgcmvXn6Gqi2S0iwaYaIHR+9vcM2qDMfHfEG7c
FgIceJZBJmRo0MlMb9JXa8ZtqwoNsiR/5agvDXQJvff/A/e8FH5ENRbUSb2HPHog
AZO0GoJgyETaczXSNztybT8+aQOMmmI1JWYd1nOMGgLp3mJgsjF4aHLbAoGBAPmG
BErH4xow6RB5Wq3Q1XPW3d+EZvxnrK5nxj1SSav66r88w7pNMEIkFlyqOZm5uIUa
2BjR03898jxSfuZeiMfSXyflaS4MpxClZKMdMdhgqhvjxhqHeMowNhKO0WX0KX8J
L6zPr6EPwrBo3zEnMcjaMhmRscUWR1DecMcKlHDdAoGBAN8OjGlXnKVRS0Pn1I1c
COHtyoDlBiezL4Gqun1zXjfHpnZ4oSuWlNf7WKYMp9jrHmKJHDZXoiKc0vycLXOc
22KJ4AHHc0FetcZ+ePYRmh+s88Dwd0cpaN7CzufEusea8TByRK53rvm+j2gIN/Ao
JB6PvjTGKKqyxaGVTUUPfHgDAoGAErOovrIco2nnDgUKdtygIv6Hwqj5zxE2MBw3
D4GLZAh6b7ruMJh4dXye8HMRviPdYJySdcnEQFU0QrEsMbgEKHXsC+F18K2iF+1N
jawygDU+iriXsIVW2FCkvN9XcnzKX2sg16L5VukHfpFdqSF26cbw2lnBKTRyQ+1o
JoL0fUECgYB8YeqAJDoG/PqAFZiXvVnCMxlIquZ3kW3hmAMwXRCErh+swb9EX5Zg
SkVZzmd5UVFUarIxnidGvzwluIpq/5ff3EW63qzMtlz259Bo7TnJJlkGPVaKVoD2
ChQcQhXht9/PJ4oqk/0iZCkcnF/xwRLhvIECymFh/dwTdvimaV/Qcg==
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
		Info.Println("Using TLS")
		path := os.Getenv("DOCKER_CERT_PATH")
		ca := fmt.Sprintf("%s/ca.pem", path)
		cert := fmt.Sprintf("%s/cert.pem", path)
		key := fmt.Sprintf("%s/key.pem", path)
		client, err := docker.NewTLSClient(endpoint, cert, key, ca)
		if err != nil {
			Error.Println("Something is wrong with your docker environemnt.")
			Trace.Println("Check: DOCKER_HOST=", endpoint)
		}
		return client
	} else {
		Info.Println("TLS is disabled")
		client, _ := docker.NewClient(endpoint)
		return client
	}
}

func startEager(client *docker.Client, image string, containerName string, data string, gatk string, uid bool) error {
	exposedCadvPort := map[docker.Port]struct{}{"22/tcp": {}}

	uenv := []string{}
	if uid {
		userobj,err	:= user.Current()
		if err !=  nil {
			fmt.Println(err)
			os.Exit(1)
		}
		uenv = []string{fmt.Sprintf("DCKR_UID=%s", userobj.Uid), fmt.Sprintf("DCKR_GID=%s", userobj.Gid)}
	}
	createContConf := docker.Config{
		ExposedPorts: exposedCadvPort,
		Image:        image,
		Env: 		  uenv,
	}

	portBindings := map[docker.Port][]docker.PortBinding{
		"22/tcp": {{HostIP: "0.0.0.0", HostPort: "2222"}}}

	gatkPath, _ := checkForGatk(gatk)
	gatkBind := fmt.Sprintf("%s:/gatk/", gatkPath)
	dataBind := fmt.Sprintf("%s:/data/", data)
	createContHostConfig := docker.HostConfig{
		// Figure out where gatk is located and add it to the bind-mounts
		Binds:           []string{gatkBind, dataBind},
		PortBindings:    portBindings,
		PublishAllPorts: false,
		Privileged:      true,
	}

	opts := docker.PullImageOptions{
		Repository: "apeltzer/eager",
	}
	err := client.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		fmt.Println(err)
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
