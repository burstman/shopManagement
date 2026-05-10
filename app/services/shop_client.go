package services

import (
	"context"
	"fmt"
	"shopDashboard/app/db"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderItem struct {
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type Order struct {
	ID               uint        `json:"id"`
	FirstName        string      `json:"first_name"`
	LastName         string      `json:"last_name"`
	Email            string      `json:"email"`
	Total            float64     `json:"total"`
	Commission       float64     `json:"commission"`
	CommissionStatus string      `json:"commission_status"`
	OrderStatus      string      `json:"order_status"`
	CreatedAt        string      `json:"created_at"`
	Items            []OrderItem `json:"items,omitempty"`
}

type Commission struct {
	TotalOrders       int64   `json:"total_orders"`
	TotalRevenue      float64 `json:"total_revenue"`
	TotalCommission   float64 `json:"total_commission"`
	PendingCommission float64 `json:"pending_commission"`
	PaidCommission    float64 `json:"paid_commission"`
}

type ShopClient struct {
	pool        *pgxpool.Pool
	affiliateID int
}

func NewShopClient(affiliateID int) *ShopClient {
	return &ShopClient{pool: db.GetPool(), affiliateID: affiliateID}
}

func (c *ShopClient) FetchOrders() ([]Order, error) {
	rows, err := c.pool.Query(context.Background(),
		`SELECT o.id, o.first_name, o.last_name, o.email,
		        o.total, o.platform_commission, o.commission_status,
		        o.status, o.created_at
		 FROM orders o
		 WHERE o.deleted_at IS NULL AND o.affiliate_id = $1
		 ORDER BY o.created_at DESC`, c.affiliateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var createdAt time.Time
		if err := rows.Scan(&o.ID, &o.FirstName, &o.LastName, &o.Email,
			&o.Total, &o.Commission, &o.CommissionStatus,
			&o.OrderStatus, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		o.CreatedAt = createdAt.Format("Jan 02, 2006 15:04")

		items, err := c.fetchOrderItems(o.ID)
		if err != nil {
			return nil, err
		}
		o.Items = items

		orders = append(orders, o)
	}
	return orders, nil
}

func (c *ShopClient) fetchOrderItems(orderID uint) ([]OrderItem, error) {
	rows, err := c.pool.Query(context.Background(),
		`SELECT product_name, quantity, price
		 FROM order_items
		 WHERE order_id = $1 AND deleted_at IS NULL`, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ProductName, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *ShopClient) FetchCommission() (*Commission, error) {
	var comm Commission
	err := c.pool.QueryRow(context.Background(),
		`SELECT COUNT(*), COALESCE(SUM(total), 0), COALESCE(SUM(platform_commission), 0)
		 FROM orders
		 WHERE deleted_at IS NULL AND affiliate_id = $1`, c.affiliateID,
	).Scan(&comm.TotalOrders, &comm.TotalRevenue, &comm.TotalCommission)
	if err != nil {
		return nil, fmt.Errorf("failed to query commission: %w", err)
	}

	err = c.pool.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(platform_commission), 0)
		 FROM orders
		 WHERE deleted_at IS NULL AND affiliate_id = $1 AND commission_status = 'pending'`, c.affiliateID,
	).Scan(&comm.PendingCommission)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending commission: %w", err)
	}

	err = c.pool.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(platform_commission), 0)
		 FROM orders
		 WHERE deleted_at IS NULL AND affiliate_id = $1 AND commission_status = 'paid'`, c.affiliateID,
	).Scan(&comm.PaidCommission)
	if err != nil {
		return nil, fmt.Errorf("failed to query paid commission: %w", err)
	}

	return &comm, nil
}
