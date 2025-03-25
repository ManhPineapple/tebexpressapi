package sqlmanager

import (
	"errors"
	"fmt"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"time"

	"gorm.io/gorm"
)

type ProductManager struct {
	db *gorm.DB
}

func NewProductManager(db *gorm.DB) *ProductManager {
	return &ProductManager{
		db: db,
	}
}

type ProductQueryOption struct {
	ID     int64
	IDS    []int64
	Search string
	Limit  int
	Offset int
	UserID int64
	Status int64
}

func (m *ProductManager) buildProductQuery(opts ProductQueryOption) *gorm.DB {
	db := m.db

	if opts.ID > 0 {
		db = db.Where("id=?", opts.ID)
	}

	if len(opts.IDS) > 0 {
		db = db.Where("id IN (?)", opts.IDS)
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}

	if opts.Search != "" {
		db = db.Where("name LIKE (?) OR sku = ?", fmt.Sprintf("%%%s%%", opts.Search), opts.Search)
	}

	if opts.UserID > 0 {
		db = db.Where("user_id = ?", opts.UserID)
	}

	if opts.Status > 0 {
		db = db.Where("status=?", opts.Status)
	}

	return db
}

func (m *ProductManager) CreateProductOrAddStock(product *entity.Product) (*entity.Product, error) {
	var productLog entity.ProductStockLogs

	isExist, err := m.CheckSKUExist(product.SKU, product.UserID)
	if err != nil {
		return nil, err
	}

	if isExist {
		var existingProduct entity.Product
		if err := m.db.Where("sku = ? AND user_id = ? AND status = ?", product.SKU, product.UserID, constant.StatusActive).First(&existingProduct).Error; err != nil {
			return nil, err
		}

		productLog = entity.ProductStockLogs{
			ProductID: existingProduct.ID,
			Quantity:  product.Stock,
		}
		if err := m.db.Create(&productLog).Error; err != nil {
			return nil, err
		}

		existingProduct.Stock += product.Stock
		if err := m.db.Save(&existingProduct).Error; err != nil {
			return nil, err
		}
		return &existingProduct, nil
	}

	if err := m.db.Create(product).Error; err != nil {
		return nil, err
	}

	productLog = entity.ProductStockLogs{
		ProductID: product.ID,
		Quantity:  product.Stock,
	}
	if err := m.db.Create(&productLog).Error; err != nil {
		return nil, err
	}

	return product, nil
}

func (m *ProductManager) GetProducts(opts ProductQueryOption) ([]*entity.Product, error) {
	db := m.buildProductQuery(opts)
	db = db.Order("id DESC")
	products := []*entity.Product{}

	db = db.Find(&products)
	return products, db.Error
}

func (m *ProductManager) GetProductLogs(productID int64) ([]*entity.ProductStockLogs, error) {
	var logs []*entity.ProductStockLogs
	err := m.db.Where("product_id = ?", productID).Order("updated_at DESC").Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (m *ProductManager) CountProducts(opts ProductQueryOption) (int64, error) {
	var count int64

	db := m.buildProductQuery(opts)

	db = db.Model(&entity.Product{}).Count(&count)
	return count, db.Error
}

func (m *ProductManager) CheckSKUExist(sku string, userID int64) (bool, error) {
	var count int64
	db := m.db.Model(&entity.Product{}).Where("sku = ? AND user_id = ? AND status = ?", sku, userID, constant.StatusActive).Count(&count)

	return count > 0, db.Error
}

func (m *ProductManager) GetProductByID(id int64) (*entity.Product, error) {
	product := &entity.Product{}
	db := m.db.Where("id = ?", id)
	db = db.First(product)
	return product, db.Error
}

func (m *ProductManager) UpdateProduct(id int64, mapchange map[string]interface{}) error {
	mapchange["UpdatedAt"] = time.Now()
	db := m.db.Model(&entity.Product{}).Where("id=?", id).UpdateColumns(mapchange)

	stockRaw, exists := mapchange["stock"]
	if exists {
		stockValue, ok := stockRaw.(int64)
		if !ok {
			return errors.New("invalid stock value type")
		}
		productLog := entity.ProductStockLogs{
			ProductID: id,
			PackageID: constant.PackageProductLogTypeUserSet,
			Quantity:  stockValue,
		}
		if err := m.db.Create(&productLog).Error; err != nil {
			return err
		}
	}
	return db.Error
}

func (m *ProductManager) AdjustStock(productID int64, packageID int64, quantity int64) error {
	db := m.db.Model(&entity.Product{}).
		Where("id = ? AND (stock + ?) >= 0", productID, quantity).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity))

	if db.RowsAffected == 0 {
		return errors.New("insufficient stock or product not found")
	}
	productLog := entity.ProductStockLogs{
		ProductID: productID,
		PackageID: packageID,
		Quantity:  quantity,
	}
	if err := m.db.Create(&productLog).Error; err != nil {
		return err
	}
	return db.Error
}
