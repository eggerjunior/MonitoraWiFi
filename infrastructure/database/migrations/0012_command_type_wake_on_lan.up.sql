-- Fase 5: Wake-on-LAN (wake_on_lan) como novo tipo de comando sob demanda.
ALTER TABLE agent_commands DROP CONSTRAINT agent_commands_type_check;
ALTER TABLE agent_commands ADD CONSTRAINT agent_commands_type_check
    CHECK (type IN ('ping', 'dns_lookup', 'traceroute', 'batch_ping', 'ssl_check', 'http_request', 'lan_scan', 'wake_on_lan'));
