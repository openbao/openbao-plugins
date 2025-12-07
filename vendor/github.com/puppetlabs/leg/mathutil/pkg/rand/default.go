package rand

// DefaultFactory is a PCG RNG that is initially seeded by the SecureSeeder.
var DefaultFactory = NewPCGFactory(SecureSeeder)
