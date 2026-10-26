param name string
param location string = resourceGroup().location
param tags object = {}
@minLength(1)
param principalId string
@minLength(1)
param principalName string
@description('Public IPv4 address allowed to initialize the database and run the test writer.')
@minLength(1)
param clientIp string

resource server 'Microsoft.Sql/servers@2023-08-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    minimalTlsVersion: '1.2'
    publicNetworkAccess: 'Enabled'
    administrators: {
      administratorType: 'ActiveDirectory'
      principalType: 'User'
      login: principalName
      sid: principalId
      tenantId: subscription().tenantId
      azureADOnlyAuthentication: true
    }
  }
}

resource database 'Microsoft.Sql/servers/databases@2023-08-01' = {
  parent: server
  name: 'quickstart'
  location: location
  tags: tags
  sku: {
    name: 'Basic'
    tier: 'Basic'
    capacity: 5
  }
  properties: {
    maxSizeBytes: 2147483648
  }
}

resource azureServices 'Microsoft.Sql/servers/firewallRules@2023-08-01' = {
  parent: server
  name: 'AllowAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

resource developer 'Microsoft.Sql/servers/firewallRules@2023-08-01' = {
  parent: server
  name: 'DeveloperClient'
  properties: {
    startIpAddress: clientIp
    endIpAddress: clientIp
  }
}

output fqdn string = server.properties.fullyQualifiedDomainName
output databaseName string = database.name
