#!groovy
pipeline {
  agent {
      docker {
          image 'golang:1.8'
      }
  }
  stages {
    stage('buildIC'){
      environment {
        version = "0.9.0"   
      }
      steps {
        sh ('go version')
        sh('cd nginx-controller && go build -a -installsuffix cgo -ldflags "-w -X main.version=${version}" -o nginx-ingress *.go')
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
