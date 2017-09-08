pipeline {

  agent {
    label "goBuilds"
  }
  
  stages {
    stage("build") {
      steps {
        timeout(20) {
          go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o nginx-ingress *.go
        }
      }
      
    }
    
  
  post {
    // Always runs. And it runs before any of the other post conditions.
    always {
      // Let's wipe out the workspace before we finish!
      //deleteDir()
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
}
