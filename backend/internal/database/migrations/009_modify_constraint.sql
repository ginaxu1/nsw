alter table public.task_infos
drop constraint task_infos_state_check;

alter table public.task_infos
    add constraint task_infos_state_check
        check ((state)::text = ANY
    ((ARRAY ['INITIALIZED'::character varying, 'IN_PROGRESS'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[]));