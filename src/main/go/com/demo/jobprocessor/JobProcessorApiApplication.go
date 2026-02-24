package com.demo.jobprocessor;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
public class JobProcessorApiApplication {

    public static void main(String[] args) {
        SpringApplication.run(JobProcessorApiApplication.class, args);
    }

}
