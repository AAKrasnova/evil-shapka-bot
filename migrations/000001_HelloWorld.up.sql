CREATE TABLE users (
	id varchar,
	tgname varchar,
	first_name varchar,
	last_name varchar,
	language varchar,
	premium integer
);

CREATE TABLE knowledge (
	id varchar,
	name varchar,
	adder varchar,
	type varchar,
	subtype varchar,
	theme varchar,
	sphere varchar,
	link text,
	word_count integer,
	duration integer,
	language varchar
);

CREATE TABLE consumed (
	id varchar,
	user_id varchar,
	knowledge_id varchar,
	date datetime,
	ready_to_re integer,
	rate integer,
	comment text
);



