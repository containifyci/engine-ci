package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/containifyci/engine-ci/cmd"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var dockerClient *client.Client

func main() {
	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	// r.HandleFunc("/containers/{id}/start", containerAction(startContainer)).Methods("POST")
	// r.HandleFunc("/containers/{id}/stop", containerAction(stopContainer)).Methods("POST")
	r.HandleFunc("/containers/{id}/logs", streamLogs).Methods("GET")
	r.HandleFunc("/containers/{id}/inspect", containerAction(inspectContainer)).Methods("GET")

	r.HandleFunc("/containers/{type}/create", createContainer).Methods("POST")

	http.Handle("/", r)
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func createContainer(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// buildType := vars["type"]

	var buildArgs container.Build

	if err := json.NewDecoder(r.Body).Decode(&buildArgs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch buildArgs.BuildType {
	case container.GoLang:
		buildArgs = container.NewGoServiceBuild(buildArgs.App)
	case container.Maven:
		buildArgs = container.NewMavenServiceBuild(buildArgs.App)
	case container.Python:
		buildArgs = container.NewPythonServiceBuild(buildArgs.App)
	}

	c := cmd.NewCommand(buildArgs, nil)
	c.AddTarget("all", func() error {
		fmt.Println("Running build")
		// cmd.RunBuild(nil, nil)
		return nil
	})
	// addr := Start()

	// c.Run("all", container.NewBuild(&buildArgs))

	// switch buildType {
	// case "golang":
	// 	golang := golang.New(dockerClient)
	// 	golang.Run()
	// 	w.Header().Set("Content-Type", "application/json")
	// 	json.NewEncoder(w).Encode(map[string]string{"ID": golang.Container.Resp.ID})
	// default:
	// }
	http.Error(w, "Invalid build type", http.StatusBadRequest)
	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// _, err := dockerClient.ImagePull(context.Background(), req.Image, image.PullOptions{})
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// containerConfig := &container.Config{
	// 	Image:      req.Image,
	// 	Entrypoint: req.Entrypoint,
	// 	Cmd:        req.Cmd,
	// }

	// hostConfig := &container.HostConfig{}

	// containerResp, err := dockerClient.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, "")
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
}

type containerActionFunc func(context.Context, http.ResponseWriter, string) error

func containerAction(action containerActionFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		containerID := vars["id"]

		if err := action(context.Background(), w, containerID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// func startContainer(ctx context.Context, w http.ResponseWriter, id string) error {
// 	return dockerClient.ContainerStart(ctx, id, container.StartOptions{})
// }

// func stopContainer(ctx context.Context, w http.ResponseWriter, id string) error {
// 	return dockerClient.ContainerStop(ctx, id, container.StopOptions{})
// }

func inspectContainer(ctx context.Context, w http.ResponseWriter, id string) error {
	containerJSON, err := dockerClient.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	data, err := json.Marshal(containerJSON)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	return err
}

func streamLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	options := dcontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	out, err := dockerClient.ContainerLogs(context.Background(), containerID, options)
	if err != nil {
		err = conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		if err != nil {
			log.Println("Error writing error message to websocket:", err)
		}
		return
	}
	defer out.Close()

	buf := make([]byte, 4096)
	for {
		n, err := out.Read(buf)
		if err != nil && err != io.EOF {
			err = conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
			if err != nil {
				log.Println("Error writing error message to websocket:", err)
			}
			return
		}
		if n == 0 {
			break
		}

		err = conn.WriteMessage(websocket.TextMessage, buf[:n])
		if err != nil {
			break
		}
	}
}
