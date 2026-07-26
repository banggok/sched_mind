CREATE TABLE public_holidays (
    id UUID PRIMARY KEY,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    description VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT public_holidays_description_not_blank CHECK (BTRIM(description) <> '')
);

CREATE INDEX public_holidays_list_idx ON public_holidays(start_date, end_date, id);
CREATE INDEX public_holidays_status_idx ON public_holidays(end_date, start_date, id);

CREATE TABLE public_holiday_dates (
    public_holiday_id UUID NOT NULL REFERENCES public_holidays(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    PRIMARY KEY (public_holiday_id, date)
);

CREATE UNIQUE INDEX public_holiday_dates_date_uidx ON public_holiday_dates(date);
CREATE INDEX public_holiday_dates_parent_idx ON public_holiday_dates(public_holiday_id);
