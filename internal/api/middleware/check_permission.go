// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckPermission is a middleware that checks if the authenticated user has the required permission via JWT claims.
func CheckPermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permsInterface, exists := c.Get("rbac_permissions")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User permissions not found in token"})
			c.Abort()
			return
		}

		userPermissions, ok := permsInterface.([]interface{})
		if !ok {
			// Sometime parsed JSON arrays come back as []interface{}
			strPerms, strOk := permsInterface.([]string)
			if strOk {
				userPermissions = make([]interface{}, len(strPerms))
				for i, p := range strPerms {
					userPermissions[i] = p
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid permissions type in token"})
				c.Abort()
				return
			}
		}

		hasPermission := false
		for _, p := range userPermissions {
			if strP, isStr := p.(string); isStr && strP == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":               "Forbidden: You do not have the required permission",
				"required_permission": permission,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
