<!--
---
name: Azure Functions Go SQL Trigger using Azure Developer CLI
description: Process Azure SQL row changes in Go on Azure Functions Flex Consumption using managed identity.
page_type: sample
languages:
- azdeveloper
- go
products:
- azure
- azure-functions
- azure-sql-database
urlFragment: functions-quickstart-go-azd-sql
---
-->

# Azure Functions Go SQL Trigger

This sample follows the [Go timer quickstart](https://github.com/Azure-Samples/functions-quickstart-go-azd-timer)
and deploys to Linux Flex Consumption with `azd`. Go worker packages are preview
packages. The `productsChanged` function receives batches of inserts, updates,
and deletes from `dbo.Products` using Azure SQL change tracking.

```json
{"Operation":0,"Item":{"ProductId":1,"Name":"Coffee","Cost":250}}
```

Operations are `Insert` (0), `Update` (1), and `Delete` (2). Cost is an integer in
cents. Deleted rows contain **primary keys only**. The handler logs the operation
and product, and returns an error for malformed rows instead of silently ignoring
them. No `function.json` is required; the Go worker indexes the registration.

**Current platform limitation:** SQL triggers using managed identity on Flex
Consumption cannot reliably scale from zero because the external SQL scaler
cannot authenticate ([upstream issue #1235](https://github.com/Azure/azure-functions-sql-extension/issues/1235)).
This sample provisions **one Always Ready instance for `productsChanged`** so its
in-app listener can run passwordlessly. This incurs continuous baseline compute
charges, and automatic scale-out is still affected. Do not remove the instance
or rely on scale-to-zero until the upstream fix is available.

## Prerequisites

- [Go 1.25 or later](https://go.dev/dl/)
- [Azure Functions Core Tools 4.12 or later](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)
- An Azure subscription with permission to create resources, assign roles, and
  configure yourself as an Azure SQL Microsoft Entra administrator
- An interactive Microsoft Entra user account for initialization and test writes
- For local Functions execution, [Azurite](https://learn.microsoft.com/azure/storage/common/storage-use-azurite)
  for host storage; SQL itself remains a real Azure database

## Deploy

```powershell
git clone https://github.com/Azure-Samples/functions-quickstart-go-azd-sql.git
cd functions-quickstart-go-azd-sql
az login
azd auth login
azd env new sql-dev
azd env set AZURE_PRINCIPAL_ID (az ad signed-in-user show --query id -o tsv)
azd env set AZURE_PRINCIPAL_NAME (az ad signed-in-user show --query userPrincipalName -o tsv)
azd env set SQL_CLIENT_IP "<your-public-IPv4-address>"
azd up
```

Choose a region supporting both Functions Flex Consumption and Azure SQL Basic
for your subscription. If SQL capacity is restricted, select another supported
region. Use the client machine's actual outbound public IPv4 address, not its
private LAN address. If it changes, update `SQL_CLIENT_IP` and run `azd provision`
again.

Infrastructure includes:

- Entra-only SQL logical server and a `quickstart` database: **Basic, 5 DTUs,
  2 GiB**. SQL passwords are neither generated nor enabled.
- Linux Flex Consumption Function App with a user-assigned managed identity and
  one Always Ready instance for the SQL trigger.
- Identity-based host storage, Application Insights, and Log Analytics.
- A VNet/private endpoints for **host storage** by default. This does **not**
  make SQL private. For a simpler development deployment, run
  `azd env set VNET_ENABLED false` before `azd up`.

SQL uses a public endpoint with TLS. One firewall rule permits your specified
client IP. The Azure-services exception (`0.0.0.0`) permits connections from Azure
networks, including other subscriptions; database authentication still applies.
This is a development configuration, not an isolated production network.
For production, use SQL private networking and a separately managed database
administrator rather than configuring the developer as server administrator.

### Database initialization

The `postprovision` hook runs `go run ./cmd/setup` under your Azure CLI sign-in.
It is idempotent and:

1. Creates `dbo.Products` with a primary key and enables database/table change
   tracking, retaining changes for two days.
2. Creates the app's contained database user from its managed-identity **client
   ID/SID**, without Microsoft Graph directory lookup.
3. Grants `SELECT` and `VIEW CHANGE TRACKING` on Products, `CREATE TABLE` and
   `CREATE SCHEMA` for trigger state, and `ALTER`/DML on the `az_func` state
   schema. The app cannot insert/update/delete Products and is not `db_owner`.

The schema-scoped `ALTER` permission allows the extension to create its internal
state tables. Azure resource RBAC alone does not grant SQL database access.
The app uses ADO.NET managed-identity authentication with the identity's **client
ID**. SQL also requires this client/application ID for the contained-user SID;
the object/principal ID is used for Azure RBAC, not this SQL SID.

If your azd version skips hooks, run the hook explicitly **before** deploying:

```shell
azd hooks run postprovision
azd deploy
```

On an initialization failure, check your Entra administrator identity and client
firewall IP, then rerun the hook. Do not compensate by adding a SQL password,
opening the firewall to the entire Internet, or granting `db_owner` to the app.

## Insert, update, and delete a product

The included Go writer uses `ActiveDirectoryDefault` authentication through the
Go SQL driver, including your Azure CLI sign-in. It does not require SQL passwords.
Load the deployment outputs in PowerShell:

```powershell
$deployment = azd env get-values --output json | ConvertFrom-Json
$env:SQL_SERVER_FQDN = $deployment.SQL_SERVER_FQDN
$env:SQL_DATABASE = $deployment.SQL_DATABASE
go run ./cmd/write -operation insert -name Coffee -cost 250
```

The writer prints the generated `ProductId`. In the Function App's Application
Insights **Logs**, verify the insert:

```kusto
traces
| where timestamp > ago(15m)
| where message contains "product change processed"
| project timestamp, message, customDimensions
```

After observing `operation=Insert product_id=1` (substitute your actual ID), update
the row:

```shell
go run ./cmd/write -operation update -id 1 -name Espresso -cost 300
```

Wait for `operation=Update` before deleting:

```shell
go run ./cmd/write -operation delete -id 1
```

Then confirm `operation=Delete`. Allow time for host startup, change polling, and
telemetry ingestion. **Do not issue all three operations rapidly**: change
tracking coalesces changes between polls and is not an audit log. An insert and
delete before the listener processes the row may produce no change notification.

## Delivery semantics

The SQL extension maintains checkpoint and lease tables in `az_func`; do not
modify them while listeners are running. Failed batches are retried by the SQL
extension with bounded attempts, not a queue dead-letter mechanism. Persist
invalid changes separately if required. A failure after earlier rows in a batch
have been handled can repeat those rows; make production side effects idempotent.
Changes older than the retention window are not guaranteed to remain available.
See the [SQL trigger implementation details](https://github.com/Azure/azure-functions-sql-extension/blob/main/docs/TriggerBinding.md).

## Run locally

Use a dedicated development database initialized by this sample. Do not run
local and deployed listeners concurrently against the same development table
unless you intentionally want independent consumers.

```powershell
Copy-Item local.settings.example.json local.settings.json
```

Replace the SQL server hostname in `local.settings.json`. Its SQL binding uses
`Authentication=Active Directory Default`, not the deployed managed identity.
Run `az login` as the database's development Entra administrator (or explicitly
grant a separate development identity the trigger permissions described above).
Start Azurite in another terminal and run:

```shell
func start
```

The host builds the app automatically. Azurite emulates host storage only,
not Azure SQL. Keep `local.settings.json` untracked.

## Tests, redeployment, and cleanup

```shell
go test ./...
go vet ./...
azd up
azd down
```

Azure SQL Basic has ongoing charges while idle; continuous change polling is not
suited to an auto-pausing database. The Always Ready instance also has continuous
baseline compute charges. Host storage, monitoring, and optional private
networking can also incur charges. `azd down` removes the environment's resources,
including the database and all sample data.
