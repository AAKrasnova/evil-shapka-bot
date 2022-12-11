CREATE TABLE users (
	id varchar PRIMARY KEY,
       tg_id int,
	tg_username varchar UNIQUE,
	tg_first_name varchar,
	tg_last_name varchar,
	tg_language varchar
);

CREATE TABLE events (
	id varchar PRIMARY KEY,
	name varchar,
	adder varchar,
       timeAdded DATETIME,
	code varchar UNIQUE,
       FOREIGN KEY (adder) 
       REFERENCES users (id) 
              ON DELETE NO ACTION
              ON UPDATE NO ACTION
);

CREATE TABLE entries (
       id varchar PRIMARY KEY,
       event_id varchar,
       user_id varchar,
       entry varchar,
       timeAdded datetime DEFAULT (DATETIME('now')),
       drawn integer,
  FOREIGN KEY (event_id) 
  REFERENCES events (id) 
         ON DELETE NO ACTION
         ON UPDATE NO ACTION,
  FOREIGN KEY (user_id) 
  REFERENCES users (id) 
         ON DELETE NO ACTION 
         ON UPDATE NO ACTION
);