package constant

import "time"

const UserRoleCustomer = "customer"
const UserRoleAdmin = "admin"
const UserRoleAccountant = "accountant"
const UserRoleWarehouse = "warehouse"
const UserRoleSupport = "support"
const UserRoleSupportLeader = "support_leader"
const UserRoleAppraiser = "appraiser"
const UserRoleHub = "hub"
const UserRoleMarketing = "marketing"
const UserRolerBusinessManager = "business_manager"
const UserRoleSaleOperation = "sale_operation"
const UserRolerShipPartner = "ship_partner"
const UserRoleSale = "sale"

const TokenKeyUserType = "Authorization"

const UserStatusActive = 1
const UserStatusDeactive = 0
const UserStatusInactive = 2

const RedisKeyCountInCorrectPassword = "incorrect_password"
const RedisKeyCountInCorrectPasswordExpiredTime = 24 * time.Hour

const RedisKeyForgotAndResetPassword = "forgot_and_reset_password"
const RedisKeyForgotAndResetPasswordExpiredTime = 24 * time.Hour

const RedisKeyConfirmEmail = "confirm_email"
const RedisKeyConfirmEmailExpiredTime = 24 * time.Hour

const RedisKeyDownloadOrder = "download_order"
const RedisKeyDownloadOrderExpiredTime = 30 * 24 * time.Hour

const RedisKeyCheckShopImport = 5 * time.Minute

const RedisKeySupplierVNNumberItem = "supplier_vn_number_item"
const RedisKeySupplierVNNumberItemExpiredTime = 90 * 24 * time.Hour

const SessionTokenAmazonExpiredTime = 900

const RedisKeyRateExChange = "rate_exchange"

const CheckDeactiveDay = 30
