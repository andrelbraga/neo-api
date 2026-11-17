DELETE FROM developments WHERE slug = 'boa-vista-residencial';
DELETE FROM organizations WHERE kind = 'incorporadora' AND name = 'Dwell'
  AND NOT EXISTS (SELECT 1 FROM developments d WHERE d.developer_id = organizations.id);
