CREATE TABLE logs (
    ID varchar(255),
    start_at varchar(255),
    end_at varchar(255),
    date varchar(255),
    feeling int
);

CREATE UNIQUE INDEX ID_Logs_Uniqueness ON logs(id);
