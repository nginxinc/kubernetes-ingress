#!groovy
pipeline {
  agent {
      docker {
          image 'golang:1.8'
          label 'golang1_8'
      }
  }
  stages {
    environment {
      version = "0.9.0"   
    }
    stage('buildIC'){
      steps {
        sh ('go -v')
        sh('go build -a -installsuffix cgo -ldflags "-w -X main.version=${version}" -o nginx-ingress *.go')
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
