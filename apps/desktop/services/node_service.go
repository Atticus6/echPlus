package services

import (
	"github.com/atticus6/echPlus/apps/desktop/database"
	"github.com/atticus6/echPlus/apps/desktop/models"
)

type NodeService struct{}

func (s *NodeService) CreateNode(name, token, address, serverIP string, port int64) (*models.Node, error) {

	node := &models.Node{
		Name:     name,
		ServerIP: serverIP,
		Token:    token,
		Port:     port,
		Address:  address,
	}

	if err := database.GetDB().Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil

}

func (s *NodeService) GetNodes() ([]models.Node, error) {
	var nodes []models.Node
	if err := database.GetDB().Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}
