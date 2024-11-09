CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS uuid
AS $$
BEGIN
  -- use random v4 uuid as starting point (which has the same variant we need)
  -- then overlay timestamp
  -- then set version 7 by flipping the 2 and 1 bit in the version 4 string
  RETURN encode(
    set_bit(
      set_bit(
        overlay(uuid_send(gen_random_uuid())
                placing substring(int8send(floor(extract(epoch from clock_timestamp()) * 1000)::bigint) from 3)
                from 1 for 6
        ),
        52, 1
      ),
      53, 1
    ),
    'hex')::uuid;
END
$$
LANGUAGE plpgsql
volatile;

-- Email Validiation using HTML5 regex 
CREATE EXTENSION citext;
CREATE DOMAIN email AS citext
  CHECK ( value ~ '^[a-zA-Z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$' );

-- Users table - Core user information
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    username VARCHAR(30) NOT NULL UNIQUE,
    profile_picture_url TEXT,
    bio VARCHAR(500),
    elo INTEGER NOT NULL DEFAULT 1000,
    
    CONSTRAINT username_length CHECK (char_length(username) >= 3),
    CONSTRAINT elo_range CHECK (elo >= 0)
);

-- User settings - Preferences and configurations
CREATE TABLE user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    preferred_language VARCHAR(5) NOT NULL DEFAULT 'en',
    notification_preferences JSONB NOT NULL DEFAULT '{}',
    keyboard_shortcuts JSONB NOT NULL DEFAULT '{}',
    accessibility_settings JSONB NOT NULL DEFAULT '{}',
    game_settings JSONB NOT NULL DEFAULT '{}'
);

CREATE TYPE auth_type AS ENUM ('email', 'google', 'github');

-- Authentication table - Separate from core user data
CREATE TABLE auth_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email email UNIQUE NOT NULL,
    auth_type AUTH_TYPE NOT NULL,
    auth_id TEXT NOT NULL,
    auth_data JSONB NOT NULL DEFAULT '{}',
    verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ
);

CREATE TYPE assets_type AS ENUM ('command');
CREATE TYPE acquisition_method AS ENUM ('purchase', 'reward', 'achievement');

-- User assets - Collectibles, skins, etc.
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    asset_type VARCHAR(30) NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE user_assets (
    id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset_id int NOT NULL,
    asset_type ASSETS_TYPE NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acquisition_method ACQUISITION_METHOD NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    
    UNIQUE (user_id, asset_id)
);


CREATE TYPE visibility AS ENUM ('private', 'public', 'friends');

-- Modules - Stored code module
CREATE TABLE modules (
    id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    visibility VISIBILITY NOT NULL DEFAULT 'private',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_modified TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE duel_type AS ENUM ('friendly','ranked','tournament');

-- Duels - Game matches
CREATE TABLE duels (
    id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    p1_id UUID NOT NULL REFERENCES users(id),
    p2_id UUID NOT NULL REFERENCES users(id),
    date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    winner_id UUID REFERENCES users(id),
    loser_id UUID REFERENCES users(id),
    duel_type DUEL_TYPE NOT NULL,
    p1_elo_delta INTEGER,
    p2_elo_delta INTEGER,

    CONSTRAINT different_players CHECK (p1_id != p2_id),
    CONSTRAINT valid_duel_status CHECK (status IN ('ongoing', 'completed', 'abandoned')),
    CONSTRAINT valid_outcome CHECK (
        ((winner_id IS NOT NULL AND loser_id IS NOT NULL) AND ((winner_id = p1_id AND loser_id = p2_id) OR (winner_id = p2_id AND loser_id = p1_id)))
    )
);

CREATE TYPE relationship AS ENUM ('friend', 'blocked');
CREATE TYPE relationship_status AS ENUM ('pending', 'accepted', 'rejected');

-- User relationships - Friends, blocked users, etc.
CREATE TABLE user_relationships (
    user1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relationship_type RELATIONSHIP NOT NULL,
    relationship_status RELATIONSHIP_STATUS NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (user1_id, user2_id),
    CONSTRAINT different_users CHECK (user1_id != user2_id)
);

-- User achievements
CREATE TABLE achievements (
    id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    criteria JSONB NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE user_achievements (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id bigint NOT NULL REFERENCES achievements(id),
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    progress JSONB NOT NULL DEFAULT '{}',
    
    PRIMARY KEY (user_id, achievement_id)
);

-- Create indexes
-- CREATE INDEX users_username_idx ON users(username);
-- CREATE INDEX auth_accounts_email_idx ON auth_accounts(email);
CREATE INDEX auth_accounts_user_id_idx ON auth_accounts(user_id);
-- CREATE INDEX duels_p1_idx ON duels(p1_id);
-- CREATE INDEX duels_p2_idx ON duels(p2_id);
CREATE INDEX user_assets_user_idx ON user_assets(user_id);
CREATE INDEX user_achievements_user_idx ON user_achievements(user_id);

-- Add triggers for last_modified
CREATE OR REPLACE FUNCTION update_last_modified()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_modified = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- CREATE TRIGGER update_modules_last_modified
--    BEFORE UPDATE ON modules
--    FOR EACH ROW
--    EXECUTE FUNCTION update_last_modified();