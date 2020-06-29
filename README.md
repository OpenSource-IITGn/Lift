# Lift
A lightweight software for P2P file transfer

## Compilation
To start the current work for testing:

```
go run main.go

```

## Structure
Lift is like a service and the service directory contains the files for the service package. The service structure is divided into:
1. File system Manager -  Takes care of indexing files and maintaing directory structure for the files.
2. Host Manager - It is resposible for discovering other nodes and user authetication. Any network work not related to direct file management is done here
3. L Service - This packages the managers into a single service


## Warning
You may encounter **Hazardous Coding Practices** and **BAD Documentation**
