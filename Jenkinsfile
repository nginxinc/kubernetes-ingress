#!groovy
pipeline {
  stages {
    stage('buildIC'){
      def golang = docker.image('golang')
      def version = "0.9.0"
      golang.inside {
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
