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
	Query(query string, args ...any) ([]Item, error)
}

type OrderService struct {
	db Database
}

func NewOrderService(db Database) *OrderService {
	return &OrderService{db: db}
}

// This function has an N+1 query problem - it queries the database once per order
func (s *OrderService) GetOrdersWithItems(userID int) ([]OrderWithItems, error) {
	orders, err := s.db.GetOrdersByUserID(userID)
	if err != nil {
		return nil, err
	}

	var result []OrderWithItems
	for _, order := range orders {
		// N+1 problem: This executes a separate query for each order
		items, err := s.db.GetOrderItems(order.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, OrderWithItems{
			Order: order,
			Items: items,
		})
	}

	return result, nil
}
