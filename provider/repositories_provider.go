package provider

import "abdanhafidz.com/event-booking-api/repositories"

type RepositoriesProvider interface {
	ProvideTransactor() repositories.Transactor
	ProvideUserRepository() repositories.UserRepository
	ProvideEventRepository() repositories.EventRepository
	ProvideTicketTypeRepository() repositories.TicketTypeRepository
	ProvideBookingRepository() repositories.BookingRepository
	ProvidePaymentRepository() repositories.PaymentRepository
	ProvideIdempotencyRepository() repositories.IdempotencyRepository
}

type repositoriesProvider struct {
	transactor            repositories.Transactor
	userRepository        repositories.UserRepository
	eventRepository       repositories.EventRepository
	ticketTypeRepository  repositories.TicketTypeRepository
	bookingRepository     repositories.BookingRepository
	paymentRepository     repositories.PaymentRepository
	idempotencyRepository repositories.IdempotencyRepository
}

func NewRepositoriesProvider(cfg ConfigProvider) RepositoriesProvider {
	db := cfg.ProvideDatabaseConfig().GetInstance()

	return &repositoriesProvider{
		transactor:            repositories.NewTransactor(db),
		userRepository:        repositories.NewUserRepository(db),
		eventRepository:       repositories.NewEventRepository(db),
		ticketTypeRepository:  repositories.NewTicketTypeRepository(db),
		bookingRepository:     repositories.NewBookingRepository(db),
		paymentRepository:     repositories.NewPaymentRepository(db),
		idempotencyRepository: repositories.NewIdempotencyRepository(db),
	}
}

func (r *repositoriesProvider) ProvideTransactor() repositories.Transactor {
	return r.transactor
}

func (r *repositoriesProvider) ProvideUserRepository() repositories.UserRepository {
	return r.userRepository
}

func (r *repositoriesProvider) ProvideEventRepository() repositories.EventRepository {
	return r.eventRepository
}

func (r *repositoriesProvider) ProvideTicketTypeRepository() repositories.TicketTypeRepository {
	return r.ticketTypeRepository
}

func (r *repositoriesProvider) ProvideBookingRepository() repositories.BookingRepository {
	return r.bookingRepository
}

func (r *repositoriesProvider) ProvidePaymentRepository() repositories.PaymentRepository {
	return r.paymentRepository
}

func (r *repositoriesProvider) ProvideIdempotencyRepository() repositories.IdempotencyRepository {
	return r.idempotencyRepository
}
