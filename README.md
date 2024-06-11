# nakama-downloader-module

* I assume I have to contribute to Nakama by creating a new module that will provide the ability to download a file using provided parameters.
* The Nakama documentation states that it's not recommended to fork and contribute directly. Instead, it suggests creating a new module using Nakama's runtime features.
* Since the task allowed me to use either Go or TypeScript, I decided to move forward with Go.
* I set up the module using [the following page](https://heroiclabs.com/docs/nakama/server-framework/go-runtime).

# About RPC

* It looks like that CRC32 is enough for hashing files in this case. I think this hash is necessary to just check that file was not changed between 2 invocation, so, it's not necessary to use cryptographic hash functions like SHA256.

# About database

* I checked source code of Nakama and it looks like it's not possible to add a custom migration to Nakama lifecycle (apply it by `migrate up`) because Nakama is only looking for `migrtion/sql/*.sql`.  For simplicity, I decided to hardcode a table scheme inside the module. 

# What can be improved

* Add database schema migration. Advanced solutions like Liquibase or Alembic might not fit because they require additional runtime environments (JRE or Python). It's possible to continue using `github.com/heroiclabs/sql-migrate` and, for example, create a custom `up` command. This command should probably call `migrate.go#up` first and then apply the migrations defined in the module. Additionally, if `AfterSchemaAppliedHook` is implemented in Nakama, it would be possible to apply custom migrations using the default `migrate up` command.
* Better config organization. I decided to use an existing Nakama feature (runtime env) and don't bring a separate config library. For more complicated configurations it will be necessary to add a separate config lib with yml/hocon support.