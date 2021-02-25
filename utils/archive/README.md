# stonk-lambda

Lambda function that triggers off of a dynamodb stream and writes to a data store used for future stock recommendation analytics processing.
### Build:
    
    # Download the build-lambda-zip tool
    $ go.exe get -u github.com/aws/aws-lambda-go/cmd/build-lambda-zip
    # Build application for linux
    $ GOOS=linux go build main.go
    # Zip build for lambda
    $ build-lambda-zip.exe -output main.zip main

### Architecture:
