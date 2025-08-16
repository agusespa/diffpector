//go:build ignore

package service

type Order struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Total  float64 `json:"total"`
	Status string  `json:"status"`
}

type Item struct {
	ID       int     `json:"id"`
	OrderID  int     `json:"order_id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type OrderWithItems struct {
	Order Order  `json:"order"`
	Items []Item `json:"items"`
}

type Database interface {
	GetOrdersByUserID(userID int) ([]Order, error)
	GetOrderItems(orderID int) ([]Item, error)
	GetAllOrderItems(orders []Order) (map[int][]Item, error)
	Query(query string, args ...any) ([]Item, error)
}

type OrderService struct {
	db Database
}

func NewOrderService(db Database) *OrderService {
	return &OrderService{db: db}
}

// GetOrdersWithItems loads orders with their associated items
func (s *OrderService) GetOrdersWithItems(userID int) ([]OrderWithItems, error) {
	orders, err := s.db.GetOrdersByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Batch loading of all items
	allItems, err := s.db.GetAllOrderItems(orders)
	if err != nil {
		return nil, err
	}

	result := make([]OrderWithItems, len(orders))
	for i, order := range orders {
		result[i] = OrderWithItems{
			Order: order,
			Items: allItems[order.ID],
		}
	}

	return result, nil
}

func (s *OrderService) combineOrdersAndItems(orders []Order, itemsMap map[int][]Item) []OrderWithItems {
	result := make([]OrderWithItems, len(orders))
	for i, order := range orders {
		result[i] = OrderWithItems{
			Order: order,
			Items: itemsMap[order.ID],
		}
	}
	return result
}
