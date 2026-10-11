package provider

import "abdanhafidz.com/event-booking-api/controllers"

type ControllerProvider interface {
	ProvideEventController() controllers.EventController
	ProvideBookingController() controllers.BookingController
}

type controllerProvider struct {
	eventController   controllers.EventController
	bookingController controllers.BookingController
}

func NewControllerProvider(servicesProvider ServicesProvider) ControllerProvider {
	return &controllerProvider{
		eventController:   controllers.NewEventController(servicesProvider.ProvideEventService()),
		bookingController: controllers.NewBookingController(servicesProvider.ProvideBookingService()),
	}
}

func (c *controllerProvider) ProvideEventController() controllers.EventController {
	return c.eventController
}

func (c *controllerProvider) ProvideBookingController() controllers.BookingController {
	return c.bookingController
}
