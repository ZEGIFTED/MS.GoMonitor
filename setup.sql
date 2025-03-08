-- Create ServiceMonitorConfigs table to store service configurations
CREATE TABLE ServiceMonitorConfigs (
    ServiceID INT IDENTITY(1,1) PRIMARY KEY,
    ServiceName NVARCHAR(255) NOT NULL,
    MonitorCommand NVARCHAR(MAX) NOT NULL,
    CheckInterval NVARCHAR(50) DEFAULT '*/5 * * * *',
    HealthCheckURL NVARCHAR(MAX) NULL,
    AlertWebhookURL NVARCHAR(MAX) NULL,
    EnvironmentVars NVARCHAR(MAX) NULL, -- Store as JSON
    IsActive BIT DEFAULT 1,
    CreatedAt DATETIME DEFAULT GETDATE(),
    UpdatedAt DATETIME DEFAULT GETDATE()
);

-- Create ServiceMonitorLogs table to track service currentStatus and failures
CREATE TABLE ServiceMonitorLogs (
    LogID INT IDENTITY(1,1) PRIMARY KEY,
    ServiceID INT NOT NULL,
    Status NVARCHAR(50) NOT NULL,
    FailureCount INT DEFAULT 0,
    ErrorLog NVARCHAR(MAX) NULL,
    LastChecked DATETIME NOT NULL,
    CreatedAt DATETIME DEFAULT GETDATE(),
    FOREIGN KEY (ServiceID) REFERENCES ServiceMonitorConfigs(ServiceID)
);

-- Example insert for a service configuration
INSERT INTO ServiceMonitorConfigs 
(ServiceName, MonitorCommand, CheckInterval, HealthCheckURL, AlertWebhookURL, EnvironmentVars) 
VALUES 
(
    'Web Server', 
    'systemctl is-active nginx', 
    '*/5 * * * *', 
    'http://localhost/health', 
    'https://your-webhook.com/alert',
    '{"PATH": "/usr/local/bin:/usr/bin:/bin"}'
);