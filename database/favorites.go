package db

import log "github.com/Sirupsen/logrus"

func initFavorites() {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gowncloud.favorites (" +
		"nodeid INTEGER references gowncloud.nodes, " +
		"username STRING references gowncloud.users, " +
		"unique (nodeid, username)" +
		")")
	if err != nil {
		log.Fatal("Failed to delete table 'favorites': ", err)
	}

	log.Debug("Initialized 'favorites' table")
}

// MarkNodeAsFavorite adds an entry poiting to a node and user. An  error is returned
// if the node or user doesn't exist.
func MarkNodeAsFavorite(path, user string) error {
	_, err := db.Exec("INSERT INTO gowncloud.favorites (nodeid, username) VALUES ("+
		"(SELECT nodeid FROM gowncloud.nodes WHERE path = $1), $2)", path, user)
	if err != nil {
		log.Errorf("Failed to mark node at path %v as favorite for user %v: %v", path, user, err)
		return ErrDB
	}

	return nil
}

// RemoveNodeAsFavorite removes the unique entry for a node and user from the database
// No entry is returned if the parameter combination is not present in the table -
// therefore if no error is returned, the combination of parameters is guaranteed to
// not exist after calling this function.
func RemoveNodeAsFavorite(path, user string) error {
	_, err := db.Exec("DELETE FROM gowncloud.favorites WHERE nodeid in ("+
		"SELECT nodeid FROM gowncloud.nodes WHERE path = $1) AND username = $2", path, user)
	if err != nil {
		log.Errorf("Failed to umark node at path %v for user %v as favorite: %v", path, user, err)
		return ErrDB
	}

	return nil
}

// IsFavoriteByNodeid checks if a user has favorited the node identified by nodeid
func IsFavoriteByNodeid(nodeid float64, user string) (bool, error) {
	row := db.QueryRow("SELECT COUNT(1) FROM gowncloud.favorites WHERE nodeid = $1 AND "+
		"username = $2", intFromFloat(nodeid), user)
	var count int
	err := row.Scan(&count)
	if err != nil {
		log.Error("Failed to verify if favorite record exists: ", err)
		return false, ErrDB
	}
	return count == 1, nil
}

// GetFavoritedNodes returns all the nodes favorited by the user
func GetFavoritedNodes(username string, targets []string) ([]*Node, error) {
	nodes, err := getFavoritedNodesForUser(username)
	if err != nil {
		return nil, err
	}
	for _, t := range targets {
		favNodes, err := getFavoritedNodesForGroup(username, t)
		if err != nil {
			return nil, err
		}
		// Make sure we don't add duplicates
		for _, favNode := range favNodes {
			found := false
			for _, node := range nodes {
				if favNode.ID == node.ID {
					found = true
					break
				}
			}
			if !found {
				nodes = append(nodes, favNode)
			}
		}
	}
	return nodes, nil
}

// getFavoritedNodesForTarget gets all the favorited nodes including shares and
// subnodes of shares
func getFavoritedNodesForGroup(username string, target string) ([]*Node, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE nodeid IN ( "+
		"SELECT nodeid FROM gowncloud.shares WHERE target LIKE $1 || '.' || '%' UNION "+
		"SELECT nodeid FROM gowncloud.nodes WHERE path LIKE ("+
		"SELECT path FROM gowncloud.nodes WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.shares WHERE target IN ($1 || '.' || '%'))) || '%' UNION "+
		"SELECT nodeid FROM gowncloud.nodes WHERE owner = $1) AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.favorites WHERE username = $2)", target, username)
	if err != nil {
		log.Errorf("Failed to get favorited nodes for user in group %v: %v", target, err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading favorites")
		return nil, ErrDB
	}
	defer rows.Close()
	nodes := make([]*Node, 0)
	for rows.Next() {
		node := &Node{}
		var nId int
		err = rows.Scan(&nId, &node.Owner, &node.Path, &node.Isdir, &node.MimeType, &node.Deleted)
		if err != nil {
			log.Error("Error while reading favorites")
			return nil, ErrDB
		}
		node.ID = floatFromInt(nId)
		nodes = append(nodes, node)
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the favorite rows")
		return nil, err
	}
	return nodes, nil
}

func getFavoritedNodesForUser(username string) ([]*Node, error) {
	rows, err := db.Query("SELECT * FROM gowncloud.nodes WHERE nodeid IN ( "+
		"SELECT nodeid FROM gowncloud.shares WHERE target = $1 UNION "+
		"SELECT nodeid FROM gowncloud.nodes WHERE path LIKE ("+
		"SELECT path FROM gowncloud.nodes WHERE nodeid IN ("+
		"SELECT nodeid FROM gowncloud.shares WHERE target = $1)) || '%' UNION "+
		"SELECT nodeid FROM gowncloud.nodes WHERE owner = $1) AND "+
		"nodeid IN (SELECT nodeid FROM gowncloud.favorites WHERE username = $1)", username)
	if err != nil {
		log.Errorf("Failed to get favorited nodes for user %v: %v", username, err)
		return nil, ErrDB
	}
	if rows == nil {
		log.Error("Error loading favorites")
		return nil, ErrDB
	}
	defer rows.Close()
	nodes := make([]*Node, 0)
	for rows.Next() {
		node := &Node{}
		var nId int
		err = rows.Scan(&nId, &node.Owner, &node.Path, &node.Isdir, &node.MimeType, &node.Deleted)
		if err != nil {
			log.Error("Error while reading favorites")
			return nil, ErrDB
		}
		node.ID = floatFromInt(nId)
		nodes = append(nodes, node)
	}
	err = rows.Err()
	if err != nil {
		log.Error("Error while reading the favorite rows")
		return nil, err
	}
	return nodes, nil
}
