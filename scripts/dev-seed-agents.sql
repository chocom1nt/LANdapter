-- DEVELOPMENT ONLY: deterministic fake clients for UI/TUI smoke testing.
-- Fake clients are always offline: they have no active WebSocket agent.
-- Run explicitly with `make seed-demo-agents`; never run in production.
INSERT INTO clients (id, hostname, os, mac, online, last_seen) VALUES
('d0000000-0000-4000-8000-000000000001', 'DEMO-WIN-11', 'Windows 11', '02:00:00:00:00:01', FALSE, NOW()),
('d0000000-0000-4000-8000-000000000002', 'DEMO-WIN-10', 'Windows 10', '02:00:00:00:00:02', FALSE, NOW() - INTERVAL '30 seconds'),
('d0000000-0000-4000-8000-000000000003', 'DEMO-UBUNTU', 'Ubuntu 24.04', '02:00:00:00:00:03', FALSE, NOW() - INTERVAL '1 minute'),
('d0000000-0000-4000-8000-000000000004', 'DEMO-DEBIAN', 'Debian 12', '02:00:00:00:00:04', FALSE, NOW() - INTERVAL '2 hours'),
('d0000000-0000-4000-8000-000000000005', 'DEMO-MACOS', 'macOS', '02:00:00:00:00:05', FALSE, NOW() - INTERVAL '1 day'),
('d0000000-0000-4000-8000-000000000006', 'DEMO-EMPTY-MAC', 'Windows Server 2022', '', FALSE, NOW() - INTERVAL '3 days')
ON CONFLICT (id) DO UPDATE SET
  hostname = EXCLUDED.hostname,
  os = EXCLUDED.os,
  mac = EXCLUDED.mac,
  online = EXCLUDED.online,
  last_seen = EXCLUDED.last_seen;
