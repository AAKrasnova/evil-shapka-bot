CREATE TABLE users (
	id varchar PRIMARY KEY,
       tg_id int,
	tg_username varchar UNIQUE,
	tg_first_name varchar,
	tg_last_name varchar,
	tg_language varchar
);

CREATE TABLE knowledge (
	id varchar PRIMARY KEY,
	name varchar,
	adder varchar,
       timeAdded DATETIME,
	type varchar,
	subtype varchar,
	theme varchar,
	sphere varchar,
	link text NOT NULL,
	word_count integer,
	duration integer,
	language varchar,
       deleted integer,
       FOREIGN KEY (adder) 
       REFERENCES users (id) 
              ON DELETE NO ACTION
              ON UPDATE NO ACTION
);

CREATE TABLE consumed (
  id varchar PRIMARY KEY,
  knowledge_id varchar,
  user_id varchar,
  date datetime,
  ready_to_re integer,
  rate integer,
  attentionRate integer,
  comment text,
  FOREIGN KEY (knowledge_id) 
  REFERENCES knowledge (id) 
         ON DELETE NO ACTION
         ON UPDATE NO ACTION,
  FOREIGN KEY (user_id) 
  REFERENCES users (id) 
         ON DELETE NO ACTION 
         ON UPDATE NO ACTION
);