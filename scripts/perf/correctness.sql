-- Run by correctness.sh. psql variables cannot be read inside a DO block, so
-- they are passed through session settings.
SELECT set_config('perf.events', :'events', false),
       set_config('perf.flash_tt', :'flash_tt', false),
       set_config('perf.flash_tickets', :'flash_tickets', false)
\gset ignored_

\echo 'Inventory per ticket type:'
SELECT t.id AS ticket_type, t.capacity, t.available,
       COALESCE(SUM(b.quantity) FILTER (WHERE b.status IN ('PENDING_PAYMENT', 'CONFIRMED')), 0) AS held,
       COUNT(b.id) FILTER (WHERE b.status = 'CONFIRMED') AS confirmed_bookings
FROM ticket_types t
LEFT JOIN bookings b ON b.ticket_type_id = t.id
WHERE t.event_id = ANY (string_to_array(current_setting('perf.events'), ',')::uuid[])
GROUP BY t.id
ORDER BY t.id;

DO $$
DECLARE
  run_events uuid[] := string_to_array(current_setting('perf.events'), ',')::uuid[];
  bad int;
  flash_available int;
  flash_active int;
BEGIN
  SELECT count(*) INTO bad FROM (
    SELECT t.id
    FROM ticket_types t
    LEFT JOIN bookings b ON b.ticket_type_id = t.id
    WHERE t.event_id = ANY (run_events)
    GROUP BY t.id, t.capacity, t.available
    HAVING t.available < 0
        OR t.capacity <> t.available + COALESCE(SUM(b.quantity) FILTER (
             WHERE b.status IN ('PENDING_PAYMENT', 'CONFIRMED')), 0)
  ) v;
  IF bad > 0 THEN
    RAISE EXCEPTION 'inventory invariant violated for % ticket type(s)', bad;
  END IF;

  SELECT count(*) INTO bad FROM (
    SELECT p.booking_id
    FROM payments p
    JOIN bookings b ON b.id = p.booking_id
    WHERE b.event_id = ANY (run_events) AND p.status = 'SUCCEEDED'
    GROUP BY p.booking_id
    HAVING count(*) > 1
  ) v;
  IF bad > 0 THEN
    RAISE EXCEPTION '% booking(s) were paid more than once', bad;
  END IF;

  SELECT count(*) INTO bad
  FROM bookings b
  WHERE b.event_id = ANY (run_events)
    AND b.status = 'PENDING_PAYMENT'
    AND b.expires_at < now() - interval '15 seconds';
  IF bad > 0 THEN
    RAISE EXCEPTION '% expired hold(s) were not released within 15 s', bad;
  END IF;

  IF current_setting('perf.flash_tt') <> '' THEN
    SELECT t.available,
           COALESCE(SUM(b.quantity) FILTER (WHERE b.status IN ('PENDING_PAYMENT', 'CONFIRMED')), 0)
      INTO flash_available, flash_active
    FROM ticket_types t
    LEFT JOIN bookings b ON b.ticket_type_id = t.id
    WHERE t.id = current_setting('perf.flash_tt')::uuid
    GROUP BY t.id, t.available;
    IF flash_available <> 0 OR flash_active <> current_setting('perf.flash_tickets')::int THEN
      RAISE EXCEPTION 'flash sale: % tickets held and % available, expected % held and 0 available',
        flash_active, flash_available, current_setting('perf.flash_tickets');
    END IF;
  END IF;
END $$;
