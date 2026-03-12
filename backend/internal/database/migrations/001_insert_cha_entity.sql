-- ============================================================================
-- Seed: Customs House Agents (CHA)
-- ============================================================================

INSERT INTO customs_house_agents (id, name, description, email)
VALUES
	('a1b2c3d4-0001-4000-8000-000000000001', 'Spectra', 'Spectra Logistics - Customs clearance and freight', 'user123@trader.dev'),
	('a1b2c3d4-0002-4000-8000-000000000002', 'Aitken Spence', 'Aitken Spence - Integrated logistics and agency services', NULL),
	('a1b2c3d4-0003-4000-8000-000000000003', 'Advantis', 'Advantis Projects - Offers experienced clearance services', NULL),
	('a1b2c3d4-0004-4000-8000-000000000004', 'Yusen', 'Yusen - Global logistics and customs', NULL),
	('a1b2c3d4-0005-4000-8000-000000000005', 'Malship', 'Malship - Shipping and customs house agency', NULL)
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email;
