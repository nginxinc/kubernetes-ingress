#!groovy
pipeline {
  agent {
      docker {
          image 'golang:1.8'
          args " -v /opt/jenkins_home:$JENKINS_HOME -v /opt/jenkins_home/workspace/o_kubernetes-ingress_master-OIS5YGUQ3T477L3GSU4BI7TV5PN5UGTBD46SZTPNXV57WRDIHDDA:/go/src/github.com/nginxinc/kubernetes-ingress "
      }
  }
  stages {
    stage('buildIC'){
      environment {
        version = "0.9.0"   
      }
      steps {
        sh 'go version'
        sh 'echo "GOROOT:$GOROOT GOPATH:$GOPATH"' 
        sh ' cd /go/src/github.com/nginxinc/kubernetes-ingress && go test ./...'
        sh ' cd /go/src/github.com/nginxinc/kubernetes-ingress && go build -a -installsuffix cgo -ldflags "-w -X main.version=${version}" -o nginx-ingress nginx-controller/*.go'
      }
    }
  }
  post {
    always {
      // Let's wipe out the workspace before we finish!
      //deleteDir()
        echo "TODO:cleanup workspace"
    }
    
    success {
        echo "Build OK"
    }

    failure {
        echo "Problems Building"
    }
  }
  options {
    buildDiscarder(logRotator(numToKeepStr:'3'))
    timeout(time: 60, unit: 'MINUTES')
  }
}
