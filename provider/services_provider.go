package provider

import (
	"abdanhafidz.com/event-booking-api/services"
)

type ServicesProvider interface {
	ProvidePaymentGatewayService() services.PaymentGatewayService
	ProvideEventService() services.EventService
	ProvideBookingService() services.BookingService
	ProvideExpiryService() services.ExpiryService
}

type servicesProvider struct {
	paymentGatewayService services.PaymentGatewayService
	eventService          services.EventService
	bookingService        services.BookingService
	expiryService         services.ExpiryService
}

func NewServicesProvider(repoProvider RepositoriesProvider, configProvider ConfigProvider) ServicesProvider {
	bookingConfig := configProvider.ProvideBookingConfig()

	paymentGatewayService := services.NewMockPaymentGatewayService(bookingConfig.GetMockPayLatency(), bookingConfig.GetPaymentTimeout())
	eventService := services.NewEventService(repoProvider.ProvideEventRepository())
	bookingService := services.NewBookingService(
		repoProvider.ProvideTransactor(),
		repoProvider.ProvideBookingRepository(),
		repoProvider.ProvideTicketTypeRepository(),
		repoProvider.ProvidePaymentRepository(),
		repoProvider.ProvideIdempotencyRepository(),
		paymentGatewayService,
		bookingConfig,
	)
	expiryService := services.NewExpiryService(
		repoProvider.ProvideTransactor(),
		repoProvider.ProvideBookingRepository(),
		repoProvider.ProvideTicketTypeRepository(),
	)
	return &servicesProvider{
		paymentGatewayService: paymentGatewayService,
		eventService:          eventService,
		bookingService:        bookingService,
		expiryService:         expiryService,
	}
}

func (s *servicesProvider) ProvidePaymentGatewayService() services.PaymentGatewayService {
	return s.paymentGatewayService
}

func (s *servicesProvider) ProvideEventService() services.EventService {
	return s.eventService
}

func (s *servicesProvider) ProvideBookingService() services.BookingService {
	return s.bookingService
}

func (s *servicesProvider) ProvideExpiryService() services.ExpiryService {
	return s.expiryService
}
