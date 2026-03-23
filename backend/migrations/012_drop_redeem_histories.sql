-- 012: 移除旧的兑换历史表，统一使用 credit_transactions 作为账本
DROP TABLE IF EXISTS `redeem_histories`;
