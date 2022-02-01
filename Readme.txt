selfupdatehttp

This is the server which updates its own binary with the new version.
The version v2.exe is signed up with cdsa.
In the code you can find public key and signature required for verification.

How to test:

To run unit tests : go test -v

Running on windows:

Run the v1.exe file
Run http://localhost:8080/check fro the browser -> it will show the current version
Click update or run http://localhost:8080/install -> it will show if install is successful
Close server -> unfortunatelly i wasn't be able to finish the auto restart.
Run the v1.exe file
Run http://localhost:8080/check -> the version should be v2

To run again:

After running the upgrade current executable becomes v2.
To test it once again we need to build the v1 calling: go build -o v1.exe


Any feedback and/or code comments will be much appreciated!

Thanks