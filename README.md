# Overview

* I assume I need to contribute to Nakama by creating a new module that will provide the ability to download a file using the provided parameters.
* The Nakama documentation states that it is not recommended to fork and contribute directly. Instead, it suggests creating a new module using Nakama's runtime features.
* Since the task allowed me to use either Go or TypeScript, I decided to proceed with Go.
* I set up the module using [the following page](https://heroiclabs.com/docs/nakama/server-framework/go-runtime).

# How to test

## Manual testing

* Build `docker compose build --no-cache`
* Run `docker compose up`
* In browser, open `http://localhost:7351/`
* Navigate, `Runtime Modules -> API Explorer`
* In the dropdown, choose `filedownloader`
* Provide necessary payload and click `Send request`

Notes:
* The `test_data` folder is mounted to the Docker container. All files placed in this folder can be queried by the RPC.
* The task specified that files should be searched by the path `\<type>\<version>.json`. I provided the ability to configure a root path for the files (see the `test.env` file). Because Docker does not allow mounting a folder to `/`, I changed `/` to `/data`. Thus, the files are located inside the container at the path `\data\<type>\<version>.json`.

## Run autotests

In the project root folder run `go test -v`

# About RPC

* It looks like CRC32 is sufficient for hashing files in this case. I believe this hash is necessary only to check that a file has not changed between two invocations, so it is not necessary to use cryptographic hash functions like SHA256.

# About database

* I checked the source code of Nakama, and it looks like it's not possible to add a custom migration to Nakama's lifecycle (apply it by `migrate up`) because Nakama only looks for `migration/sql/*.sql`. For simplicity, I decided to hardcode the table schema inside the module. 
* Because there were no specific requirements regarding what I should store in the database, I decided to store statistics of file queries.

# What can be improved

* Add database schema migration. Advanced solutions like Liquibase or Alembic might not be suitable because they require additional runtime environments (JRE or Python). It's possible to continue using `github.com/heroiclabs/sql-migrate` and, for example, create a custom `up` command. This command should probably call `migrate.go#up` first and then apply the migrations defined in the module. Additionally, if AfterSchemaAppliedHook is implemented in Nakama, it would be possible to apply custom migrations using the default `migrate up` command.
* Better config organization: because I only need to add three properties to the config, I decided to put them in a Docker environment file. For complex applications, it might be necessary to use a config management library that provides the ability to build the config using files, environment variables, and command-line arguments.