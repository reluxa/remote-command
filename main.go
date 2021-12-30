package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"os/exec"

	"github.com/dotcypress/phonetics"
	"github.com/gin-gonic/gin"
)

type RemoteCommand struct {
	Action  string `json:"action"`
	Created string `json:"createdAt"`
}

type Task struct {
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	Executable  string   `json:"executable"`
	Arguments   []string `json:"arguments"`
}

type AliasWithDistance struct {
	Alias    string `json:"alias"`
	Task     Task   `json:"task"`
	Distance int    `json:"distance"`
}

func (a AliasWithDistance) String() string {
	b, _ := json.Marshal(a)
	return string(b)
}

func (t Task) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}

func loadTasks() []Task {
	log.SetOutput(os.Stdout)
	jsonFile, err := ioutil.ReadFile("task.json")
	if err != nil {
		log.Fatal(err)
	}

	var tasks []Task

	err = json.Unmarshal(jsonFile, &tasks)
	if err != nil {
		log.Fatal(err)
	}

	return tasks
}

func calculateDistanceOfCommand(command1 string, command2 string) int {
	words1 := strings.Fields(command1)
	words2 := strings.Fields(command2)

	// when the length of the two strings are not matching
	if len(words1) != len(words2) {
		return 0
	}

	// when the length is the same then we just return the average of the matching
	sum := 0
	for i := range words1 {
		sum += phonetics.DifferenceSoundex(words1[i], words2[i])
	}

	log.Println("Distance of '" + command1 + "' and '" + command2 + "' is " + strconv.Itoa(sum/len(words1)))
	return sum / len(words1)
}

func searchTask(tasks []Task, command string) *Task {
	var maxItem *AliasWithDistance
	for _, task := range tasks {
		for _, alias := range task.Aliases {
			distance := calculateDistanceOfCommand(alias, command)
			if maxItem == nil || alias == command || maxItem.Distance < distance {
				maxItem = &AliasWithDistance{
					Alias:    alias,
					Distance: distance,
					Task:     task,
				}
			}
		}
	}
	if maxItem != nil {
		log.Println("Task Found: " + maxItem.String())
	}
	return &maxItem.Task
}

func isInToleranceWindow(commandTime string) bool {
	t, err := time.Parse("January 02, 2006 at 03:04PM", commandTime)
	if err != nil {
		log.Println(err)
		return false
	}
	return math.Abs(float64(time.Now().Unix()-t.Unix())) < 120
}

func main() {
	r := gin.Default()
	tasks := loadTasks()

	// For each matched request Context will hold the route definition
	r.POST("/remote", func(c *gin.Context) {
		body, _ := ioutil.ReadAll(c.Request.Body)
		log.Println("Received request: " + string(body))
		var remoteCommand RemoteCommand
		json.Unmarshal(body, &remoteCommand)
		task := searchTask(tasks, remoteCommand.Action)
		if task != nil && isInToleranceWindow(remoteCommand.Created) {
			log.Println("Received: '" + remoteCommand.Action + "', executing '" + task.Description + "' command")
			executeTask(task)
			c.JSON(http.StatusOK, gin.H{"status": "OK"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"status": "Command Not Found"})
		}

	})
	r.Run(":22551")
}

func executeTask(task *Task) {
	_, err := exec.Command(task.Executable, task.Arguments...).Output()
	if err != nil {
		log.Print(err.Error())
	}
}
