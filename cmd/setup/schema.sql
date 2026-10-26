IF NOT EXISTS (SELECT 1 FROM sys.change_tracking_databases WHERE database_id = DB_ID())
    ALTER DATABASE CURRENT SET CHANGE_TRACKING = ON
    (CHANGE_RETENTION = 2 DAYS, AUTO_CLEANUP = ON);

IF OBJECT_ID(N'dbo.Products', N'U') IS NULL
    CREATE TABLE dbo.Products (
        ProductId int IDENTITY(1,1) NOT NULL PRIMARY KEY,
        Name nvarchar(100) NOT NULL,
        Cost int NOT NULL CHECK (Cost >= 0)
    );

IF NOT EXISTS (SELECT 1 FROM sys.change_tracking_tables WHERE object_id = OBJECT_ID(N'dbo.Products'))
    ALTER TABLE dbo.Products ENABLE CHANGE_TRACKING;

IF SCHEMA_ID(N'az_func') IS NULL
    EXEC(N'CREATE SCHEMA az_func');

DECLARE @sid varbinary(16) = CONVERT(varbinary(16), CONVERT(uniqueidentifier, @clientId));
IF EXISTS (SELECT 1 FROM sys.database_principals WHERE name = @identityName AND sid <> @sid)
    THROW 50001, 'Existing SQL user does not match the app managed identity.', 1;

DECLARE @quotedName nvarchar(258) = QUOTENAME(@identityName);
IF @quotedName IS NULL
    THROW 50002, 'Invalid managed identity name.', 1;

IF NOT EXISTS (SELECT 1 FROM sys.database_principals WHERE name = @identityName)
BEGIN
    DECLARE @createUser nvarchar(max) =
        N'CREATE USER ' + @quotedName + N' WITH SID = ' +
        CONVERT(nvarchar(34), @sid, 1) + N', TYPE = E;';
    EXEC(@createUser);
END;

DECLARE @grants nvarchar(max) =
    N'GRANT SELECT, VIEW CHANGE TRACKING ON dbo.Products TO ' + @quotedName + N';' +
    N'GRANT CREATE TABLE, CREATE SCHEMA TO ' + @quotedName + N';' +
    N'GRANT ALTER, SELECT, INSERT, UPDATE, DELETE ON SCHEMA::az_func TO ' + @quotedName + N';';
EXEC(@grants);
