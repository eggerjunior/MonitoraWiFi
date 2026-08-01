-- Fase 5: ping em lote (batch_ping) como novo tipo de comando sob demanda.
ALTER TABLE agent_commands DROP CONSTRAINT agent_commands_type_check;
ALTER TABLE agent_commands ADD CONSTRAINT agent_commands_type_check
    CHECK (type IN ('ping', 'dns_lookup', 'traceroute', 'batch_ping'));
