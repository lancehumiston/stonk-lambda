# stonk-lambda

Lambda function that pushes data to an SNS topic when a stock from Robinhood's "Top Movers" list reaches a configured regular market gain percentage threshold.
### Build:
    
    # Download the build-lambda-zip tool
    $ go.exe get -u github.com/aws/aws-lambda-go/cmd/build-lambda-zip
    # Build application for linux
    $ GOOS=linux go build main.go
    # Zip build for lambda
    $ build-lambda-zip.exe -output main.zip main

### Architecture:
![diagram](docs/diagram.png)
