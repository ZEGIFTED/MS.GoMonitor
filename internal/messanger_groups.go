package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"strings"

	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

func FetchUsersAndGroupsByServiceNames(ctx context.Context, db *sql.DB, systemMonitorIds []string, serviceNames []string) (map[string]mstypes.NotificationRecipients, error) {
	// Initialize the result map
	recipientMap := make(map[string]mstypes.NotificationRecipients)

	// Convert service IDs to a comma-separated string
	serviceIDsString := strings.Join(serviceNames, ",")
	systemMonitorIdString := strings.Join(systemMonitorIds, ",")

	log.Printf("Fetching recipients for service: %s. %s", serviceIDsString, systemMonitorIdString)

	// Construct the query dynamically
	//query := "EXEC NotificationGroupsSP @NOTIFY_SERVICE_GROUP = 'MONITOR_SERVICE', @APP_OR_SERVICE_IDs = ?, @SERVICE_NAMES = ?;"
	//
	//// Execute the query
	//rows, err := db.QueryContext(ctx, query, serviceIDsString, systemMonitorIdString)
	query := "EXEC NotificationGroupsSP @NOTIFY_SERVICE_GROUP = 'MONITOR_SERVICE', @SERVICE_NAMES = @serviceNames, @APP_OR_SERVICE_IDs = @systemMonitorIds;"

	rows, err := db.QueryContext(ctx, query,
		sql.Named("serviceNames", serviceIDsString),
		sql.Named("systemMonitorIds", systemMonitorIdString),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err.Error())
	}

	defer func(rows *sql.Rows) {
		err_ := rows.Close()
		if err_ != nil {

		}
	}(rows)

	var allRecipients []mstypes.NotificationRecipient

	for rows.Next() {
		var r mstypes.NotificationRecipient
		err_ := rows.Scan(
			&r.SystemMonitorId,
			&r.ServiceName,
			&r.UserName,
			&r.Email,
			&r.PhoneNumber,
			&r.SlackId,
			&r.GroupName,
			&r.Platform,
		)

		if err_ != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		slog.Debug("")

		allRecipients = append(allRecipients, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Populate the recipientMap
	for _, r := range allRecipients {
		// Check if the SystemMonitorId already exists in the map
		recipients, exists := recipientMap[r.SystemMonitorId.String()+"|"+r.ServiceName]
		if !exists {
			recipientMap[r.SystemMonitorId.String()+"|"+r.ServiceName] = mstypes.NotificationRecipients{
				Users: []mstypes.NotificationRecipient{},
			}
		}

		//log.Printf("Recipient: %s......%v", r.SystemMonitorId.String()+"|"+r.ServiceName, r)

		// Append the recipient to the Users slice
		recipients.Users = append(recipients.Users, r)

		// Reassign the updated struct back to the map
		recipientMap[r.SystemMonitorId.String()+"|"+r.ServiceName] = recipients
	}

	return recipientMap, nil
}

func FetchReportRecipients(db *sql.DB) (map[string]mstypes.NotificationRecipients, error) {
	return make(map[string]mstypes.NotificationRecipients), nil
}

// GroupRecipientsByPlatform Helper function to group recipients by notification platform
func GroupRecipientsByPlatform(recipients []mstypes.NotificationRecipient) map[string][]mstypes.NotificationRecipient {
	platformGroups := make(map[string][]mstypes.NotificationRecipient)

	for _, recipient := range recipients {
		platformGroups[recipient.Platform] = append(
			platformGroups[recipient.Platform],
			recipient,
		)
	}

	return platformGroups
}
