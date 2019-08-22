package testflight_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Admin priviliges", func() {
	var tmpDir, ogHome, oldTarget, pipelineName, pipelineConfig, resourceName, resourceTypeName, jobName, taskName string
	priviligedAdminTarget := testflightFlyTarget + "-padmin"
	newTeamName := "priviliged-admin-test-team"

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "fly-test")
		Expect(err).ToNot(HaveOccurred())

		ogHome = os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)

		Eventually(func() *gexec.Session {
			login := spawnFlyLogin(adminFlyTarget)
			<-login.Exited
			return login
		}, 2*time.Minute, time.Second).Should(gexec.Exit(0))

		pipelineName = randomPipelineName()
		resourceName = "time-test-resource"
		resourceTypeName = "my-time"
		jobName = "admin-sample-job"
		taskName = "simple-task"

		pipelineConfig = filepath.Join(tmpDir, "pipeline.yml")

		err = ioutil.WriteFile(pipelineConfig,
			[]byte(`---
resource_types:
- name: `+resourceTypeName+`
  type: registry-image
  source: {repository: concourse/time-resource}

resources:
- name: `+resourceName+`
  type: my-time
  source: {interval: 1h}

jobs:
  - name: `+jobName+`
    public: true
    plan:
      - get: `+resourceName+`
      - task: `+taskName+`
        config:
          platform: linux
          image_resource:
            type: registry-image
            source: { repository: busybox }
          run:
            path: /bin/sh
            args:
            - -c
            - |
              until test -f /tmp/stop-waiting; do
                echo 'waiting for /tmp/stop-waiting to exist'
                sleep 1
              done
              sleep 100
              echo done
              `), 0644)
		Expect(err).NotTo(HaveOccurred())

		oldTarget = flyTarget
		flyTarget = adminFlyTarget
		fly("set-team", "--non-interactive", "-n", newTeamName, "--local-user", "guest")
		wait(spawnFlyLogin(priviligedAdminTarget, "-n", newTeamName))
		flyTarget = priviligedAdminTarget
	})

	AfterEach(func() {
		fly("-t", priviligedAdminTarget, "destroy-team", "--non-interactive", "-n", newTeamName)
		os.RemoveAll(tmpDir)
		os.Setenv("HOME", ogHome)
		flyTarget = oldTarget
	})

	Context("Team-scoped commands", func() {
		It("Admin user is able to run fly execute on a team", func() {
			err := ioutil.WriteFile(
				filepath.Join(tmpDir, "task.yml"),
				[]byte(`---
        platform: linux

        image_resource:
          type: registry-image
          source: { repository: busybox }

        run:
          path: /bin/sh
          args:
          - -c
          - |
            echo done
      `),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())
			fly("execute", "-c", filepath.Join(tmpDir, "task.yml"))
		})

		It("Admin user is able to perform all team-scoped commands", func() {
			fly("set-pipeline", "--non-interactive", "-p", pipelineName, "-c", pipelineConfig)
			fly("unpause-pipeline", "-p", pipelineName)
			fly("pause-job", "-j", pipelineName+"/"+jobName)
			fly("unpause-job", "-j", pipelineName+"/"+jobName)

			fly("expose-pipeline", "-p", pipelineName)
			fly("hide-pipeline", "-p", pipelineName)

			sess := fly("get-pipeline", "-p", pipelineName)
			Expect(sess.Out.Contents()).To(ContainSubstring("echo"))

			fly("check-resource", "-r", pipelineName+"/"+resourceName)

			sess = fly("resource-versions", "-r", pipelineName+"/"+resourceName)
			Expect(sess.Out.Contents()).To(ContainSubstring("time:20"))

			sess = fly("order-pipelines", "-p", pipelineName)
			Expect(sess.Out.Contents()).To(ContainSubstring("ordered pipelines"))
			Expect(sess.Out.Contents()).To(ContainSubstring(pipelineName))

			sess = fly("jobs", "-p", pipelineName)
			Expect(sess.Out.Contents()).To(ContainSubstring("admin-sample-job"))

			sess = fly("resources", "-p", pipelineName)
			Expect(sess.Out.Contents()).To(ContainSubstring(resourceName))

			fly("trigger-job", "-j", pipelineName+"/"+jobName)
			watchSess := spawnFly("watch", "-j", pipelineName+"/"+jobName)

			sess = fly("containers")
			Expect(sess.Out.Contents()).To(ContainSubstring("check"))

			sess = fly("volumes")
			Expect(sess.Out.Contents()).To(ContainSubstring("resource-type"))

			sess = fly("check-resource-type", "-r", pipelineName+"/"+resourceTypeName)
			Expect(sess.Out.Contents()).To(ContainSubstring("checked"))

			sess = fly("checklist", "-p", pipelineName)
			Expect(sess.Out.Contents()).To(ContainSubstring(jobName))

			fly("abort-build", "-j", pipelineName+"/"+jobName, "-b", "1")
			<-watchSess.Exited
			Expect(watchSess).To(gexec.Exit(3))

			sess = fly("clear-task-cache", "-n", "-j", pipelineName+"/"+jobName, "-s", taskName)
			Expect(sess.Out.Contents()).To(ContainSubstring("caches removed"))

			fly("rename-pipeline", "-o", pipelineName, "-n", pipelineName+"-new")

			fly("pause-pipeline", "-p", pipelineName+"-new")
			fly("destroy-pipeline", "--non-interactive", "-p", pipelineName+"-new")

			fly("rename-team", "-o", newTeamName, "-n", newTeamName)
		})
	})
})
